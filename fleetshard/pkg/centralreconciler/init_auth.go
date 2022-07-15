package centralreconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	centralClient "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central_client"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	acsErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	core "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralHtpasswdSecretName = "central-htpasswd"
	adminPasswordSecretKey    = "password"
	centralServiceName        = "central"
)

var (
	groupCreators = []func(providerId string, auth private.ManagedCentralAllOfSpecAuth) *storage.Group{
		func(providerId string, auth private.ManagedCentralAllOfSpecAuth) *storage.Group {
			return &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: providerId,
				},
				RoleName: "None",
			}
		},
		func(providerId string, auth private.ManagedCentralAllOfSpecAuth) *storage.Group {
			return &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: providerId,
					Key:            "userid",
					Value:          auth.OwnerUserId,
				},
				RoleName: "Admin",
			}
		},
		func(providerId string, auth private.ManagedCentralAllOfSpecAuth) *storage.Group {
			return &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: providerId,
					Key:            "groups",
					Value:          "org_admin",
				},
				RoleName: "Admin",
			}
		},
	}
)

type AuthProviderResponse struct {
	Id string `json:"id"`
}

// CreateRHSSOAuthProvider initialises sso.redhat.com auth provider in a deployed Central instance.
func CreateRHSSOAuthProvider(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) error {
	pass, err := getAdminPassword(ctx, central, client)
	if err != nil {
		return err
	}

	address, err := getServiceAddress(ctx, central, client)
	if err != nil {
		return err
	}

	// Send auth provider POST request to Central.
	authProviderRequest := createAuthProviderRequest(central)
	resp, err := centralClient.SendRequestToCentral(ctx, authProviderRequest, address+"/v1/authProviders", pass)
	if err != nil {
		return errors.Wrap(err, "sending new auth provider to central")
	} else if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "Failed to create auth provider for central %s", central.Metadata.Name)
	}

	// Decode auth provider response to use auth provider id in /v1/groups requests.
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			glog.Warningf("Attempt to close auth provider response failed: %s", err)
		}
	}()
	var authProviderResp AuthProviderResponse
	err = json.NewDecoder(resp.Body).Decode(&authProviderResp)
	if err != nil {
		return errors.Wrap(err, "decoding auth provider POST response")
	}

	// Initiate sso.redhat.com auth provider groups.
	for _, groupCreator := range groupCreators {
		group := groupCreator(authProviderResp.Id, central.Spec.Auth)
		err = sendGroupRequest(ctx, central, group, address, pass)
		if err != nil {
			return err
		}
	}
	return nil
}

func sendGroupRequest(ctx context.Context, central private.ManagedCentral, groupRequest *storage.Group, address string, pass string) error {
	resp, err := centralClient.SendRequestToCentral(ctx, groupRequest, address+"/v1/groups", pass)
	if err != nil {
		return errors.Wrap(err, "sending new group to central")
	}
	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return acsErrors.NewErrorFromHTTPStatusCode(resp.StatusCode, "Failed to create group for central %s", central.Metadata.Name)
	}
	return nil
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
func getServiceAddress(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (string, error) {
	service := &core.Service{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: centralServiceName, Namespace: central.Metadata.Namespace},
		service)
	if err != nil {
		return "", errors.Wrapf(err, "getting k8s service for central")
	}
	if len(service.Spec.Ports) == 0 {
		return "", errors.Errorf("no ports present in %s service", centralServiceName)
	}
	address := fmt.Sprintf("https://%s.%s.svc.cluster.local:%d", centralServiceName, central.Metadata.Namespace, service.Spec.Ports[0].Port)
	return address, nil
}

func getAdminPassword(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (string, error) {
	secretRef := ctrlClient.ObjectKey{
		Name:      centralHtpasswdSecretName,
		Namespace: central.Metadata.Namespace,
	}
	secret := &core.Secret{}
	err := client.Get(ctx, secretRef, secret)
	if err != nil {
		return "", errors.Wrap(err, "getting admin password secret")
	}
	password := string(secret.Data[adminPasswordSecretKey])
	if password == "" {
		return "", errors.Errorf("no password present in %s secret.", centralHtpasswdSecretName)
	}
	return password, nil
}
