package centralreconciler

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	acsErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	core "k8s.io/api/core/v1"
	"net/http"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralHtpasswdSecretName = "central-htpasswd"
	adminPasswordSecretKey    = "password"
	centralServiceName        = "central"
)

var (
	insecureTransport *http.Transport
)

func init() {
	insecureTransport = http.DefaultTransport.(*http.Transport).Clone()
	// TODO: once certificates will be added, we probably will be able to replace with secure transport
	insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

// InitRHSSOAuthProvider initialises sso.redhat.com auth provider in a deployed Central instance.
func InitRHSSOAuthProvider(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) error {
	// Acquire central admin password.
	pass, err := acquireAdminPassword(ctx, central, client)
	if err != nil {
		return err
	}

	// Acquire central address.
	address, err := acquireServiceAddress(ctx, central, client)
	if err != nil {
		return err
	}

	// Send POST request to Central.
	authProviderRequest := createAuthProviderRequest(central)
	jsonBytes, err := json.Marshal(authProviderRequest)
	if err != nil {
		return errors.Wrap(err, "marshalling new auth provider request to central")
	}
	req, err := http.NewRequest(http.MethodPost, address, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return errors.Wrap(err, "creating HTTP request to central")
	}
	req.SetBasicAuth("admin", pass)
	req = req.WithContext(ctx)

	httpClient := http.Client{
		Transport: insecureTransport,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending new auth provider request to central")
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	} else {
		return acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "Failed to create auth provider for central %s", central.Metadata.Name)
	}
}

func createAuthProviderRequest(central private.ManagedCentral) *storage.AuthProvider {
	request := &storage.AuthProvider{
		// TODO: ROX-11619: change depending on whether environment is stage or not
		Name:       "Red Hat SSO(Stage)",
		Type:       "oidc",
		UiEndpoint: central.Spec.Endpoint.Host,
		Enabled:    true,
		Config: map[string]string{
			// TODO: ROX-11619: make configurable
			"issuer":        "https://sso.stage.redhat.com/auth/realms/redhat-external",
			"client_id":     central.Spec.Auth.ClientId,
			"client_secret": central.Spec.Auth.ClientSecret,
			"mode":          "post",
		},
		// TODO: for testing purposes only; remove once host is correctly specified in fleet-manager
		ExtraUiEndpoints: []string{"localhost:8443"},
	}
	return request
}

// TODO: ROX-11644: doesn't work when fleetshard-sync deployed outside of Central's cluster
func acquireServiceAddress(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (string, error) {
	service := &core.Service{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: centralServiceName, Namespace: central.Metadata.Namespace},
		service)
	if err != nil {
		return "", errors.Wrapf(err, "obtain k8s service for central")
	}
	if len(service.Spec.Ports) == 0 {
		return "", errors.Errorf("No ports present in %s service", centralServiceName)
	}
	address := fmt.Sprintf("https://%s.%s.svc.cluster.local:%d/v1/authProviders", centralServiceName, central.Metadata.Namespace, service.Spec.Ports[0].Port)
	return address, nil
}

func acquireAdminPassword(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (string, error) {
	secretRef := ctrlClient.ObjectKey{
		Name:      centralHtpasswdSecretName,
		Namespace: central.Metadata.Namespace,
	}
	secret := &core.Secret{}
	err := client.Get(ctx, secretRef, secret)
	if err != nil {
		return "", errors.Wrap(err, "obtaining admin password secret")
	}
	password := string(secret.Data[adminPasswordSecretKey])
	if password == "" {
		return "", errors.Errorf("No password present in %s secret. This should not be the case in Central instances installed via operator.", centralHtpasswdSecretName)
	}
	return password, nil
}
