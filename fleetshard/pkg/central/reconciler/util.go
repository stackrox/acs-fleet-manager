package reconciler

import (
	"context"
	"fmt"
	"strings"

	centralClientPkg "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/urlfmt"
	appsv1 "k8s.io/api/apps/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	core "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralServiceName = "central"
	oidcType           = "oidc"
)

func isCentralDeploymentReady(ctx context.Context, client ctrlClient.Client, central *private.ManagedCentral) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: "central", Namespace: central.Metadata.Namespace},
		deployment)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "retrieving central deployment resource from Kubernetes")
	}
	if deployment.Status.AvailableReplicas > 0 && deployment.Status.UnavailableReplicas == 0 {
		return true, nil
	}
	return false, nil
}

// TODO: ROX-11644: doesn't work when fleetshard-sync deployed outside of Central's cluster
func getServiceAddress(ctx context.Context, central *private.ManagedCentral, client ctrlClient.Client) (string, error) {
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

// authProviderName deduces auth provider name from issuer URL.
func authProviderName(central *private.ManagedCentral) (name string) {
	switch {
	case strings.Contains(central.Spec.Auth.Issuer, "sso.stage.redhat"):
		name = "Red Hat SSO (stage)"
	case strings.Contains(central.Spec.Auth.Issuer, "sso.redhat"):
		name = "Red Hat SSO"
	default:
		name = urlfmt.GetServerFromURL(central.Spec.Auth.Issuer)
	}
	if name == "" {
		name = "SSO"
	}
	return
}

// hasDefaultAuthProvider verifies whether the given central has a default auth provider as well as whether it is
// a legacy one, i.e. the default auth provider is not created via declarative config.
// It will return two booleans that indicate whether the auth provider exists and whether it's a legacy one.
func hasDefaultAuthProvider(ctx context.Context, central *private.ManagedCentral, client ctrlClient.Client) (bool, bool, error) {
	ready, err := isCentralDeploymentReady(ctx, client, central)
	if !ready || err != nil {
		return false, false, err
	}
	address, err := getServiceAddress(ctx, central, client)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, false, nil
		}
		return false, false, err
	}

	centralClient := centralClientPkg.NewCentralClientNoAuth(*central, address)
	authProvidersResp, err := centralClient.GetLoginAuthProviders(ctx)
	if err != nil {
		return false, false, errors.Wrap(err, "sending GetLoginAuthProviders request to central")
	}

	for _, provider := range authProvidersResp.AuthProviders {
		name := authProviderName(central)
		if provider.Type == oidcType {
			if provider.GetName() == name {
				// The auth provider can be considered legacy if it's UUID doesn't match the declarative config one based on
				// the name.
				return true, provider.GetId() == declarativeconfig.NewDeclarativeAuthProviderUUID(name).String(), nil
			}
		}
	}
	return false, false, nil
}
