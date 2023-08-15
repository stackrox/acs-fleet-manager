package reconciler

import (
	"context"
	"fmt"
	"strings"

	centralClientPkg "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/client"
	"github.com/stackrox/rox/pkg/urlfmt"
	appsv1 "k8s.io/api/apps/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	core "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralDeploymentName = "central"
	centralServiceName    = "central"
	oidcType              = "oidc"
)

func isCentralDeploymentReady(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: centralDeploymentName, Namespace: namespace},
		deployment)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "retrieving central deployment resource from Kubernetes")
	}
	return deployment.Status.AvailableReplicas > 0 && deployment.Status.UnavailableReplicas == 0, nil
}

// TODO: ROX-11644: doesn't work when fleetshard-sync deployed outside of Central's cluster
func getServiceAddress(ctx context.Context, namespace string, client ctrlClient.Client) (string, error) {
	service := &core.Service{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: centralServiceName, Namespace: namespace},
		service)
	if err != nil {
		return "", errors.Wrapf(err, "getting k8s service for central")
	}
	port, err := getHTTPSServicePort(service)
	if err != nil {
		return "", err
	}
	address := fmt.Sprintf("https://%s.%s.svc.cluster.local:%d", centralServiceName, namespace, port)
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
func authProviderName(central private.ManagedCentral) (name string) {
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

// hasAuthProvider verifies whether the given central has a default auth provider.
// It will return a boolean that indicates whether the auth provider exists.
func hasAuthProvider(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (bool, error) {
	ready, err := isCentralDeploymentReady(ctx, client, central.Metadata.Namespace)
	if !ready || err != nil {
		return false, err
	}
	address, err := getServiceAddress(ctx, central.Metadata.Namespace, client)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	centralClient := centralClientPkg.NewCentralClientNoAuth(central, address)
	authProvidersResp, err := centralClient.GetLoginAuthProviders(ctx)
	if err != nil {
		return false, errors.Wrap(err, "sending GetLoginAuthProviders request to central")
	}

	for _, provider := range authProvidersResp.AuthProviders {
		if provider.Type == oidcType && provider.GetName() == authProviderName(central) {
			return true, nil
		}
	}
	return false, nil
}
