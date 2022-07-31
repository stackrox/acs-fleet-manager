package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	centralClientPkg "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/ternary"
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

// createRHSSOAuthProvider initialises sso.redhat.com auth provider in a deployed Central instance.
func createRHSSOAuthProvider(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) error {
	pass, err := getAdminPassword(ctx, central, client)
	if err != nil {
		return err
	}

	address, err := getServiceAddress(ctx, central, client)
	if err != nil {
		return err
	}

	centralClient := centralClientPkg.NewCentralClient(central, address, pass)

	authProviderRequest := createAuthProviderRequest(central)
	authProviderResp, err := centralClient.SendAuthProviderRequest(ctx, authProviderRequest)
	if err != nil {
		return fmt.Errorf("sending AuthProvider request to central: %w", err)
	}

	// Initiate sso.redhat.com auth provider groups.
	for _, groupCreator := range groupCreators {
		group := groupCreator(authProviderResp.ID, central.Spec.Auth)
		err = centralClient.SendGroupRequest(ctx, group)
		if err != nil {
			return fmt.Errorf("sending group request to central: %w", err)
		}
	}
	return nil
}

func createAuthProviderRequest(central private.ManagedCentral) *storage.AuthProvider {
	request := &storage.AuthProvider{
		Name: ternary.String(strings.Contains(central.Spec.Auth.Issuer, "stage"),
			"Red Hat SSO (Stage)", "Red Hat SSO"),
		Type:       "oidc",
		UiEndpoint: central.Spec.UiEndpoint.Host,
		Enabled:    true,
		Config: map[string]string{
			"issuer":        central.Spec.Auth.Issuer,
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
	port, err := getHTTPSServicePort(service)
	if err != nil {
		return "", err
	}
	address := fmt.Sprintf("https://%s.%s.svc.cluster.local:%d", centralServiceName, central.Metadata.Namespace, port)
	return address, nil
}

func getHTTPSServicePort(service *core.Service) (int32, error) {
	for _, servicePort := range service.Spec.Ports {
		if servicePort.Name == "https" {
			return servicePort.Port, nil
		}
	}
	return 0, errors.Errorf("no `https` port is present in %s/%s service", service.Namespace, service.Name)
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
		return "", errors.Errorf("no password present in %s secret", centralHtpasswdSecretName)
	}
	return password, nil
}
