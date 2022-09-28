package k8s

import (
	"context"
	"fmt"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralReencryptRouteName   = "managed-central-reencrypt"
	centralPassthroughRouteName = "managed-central-passthrough"
	centralTLSSecretName        = "central-tls" // pragma: allowlist secret
)

// ErrCentralTLSSecretNotFound returned when central-tls secret is not found during creation of the reencrypt route
var ErrCentralTLSSecretNotFound = errors.New("central-tls secret not found")

// RouteService is responsible for performing read and write operations on the OpenShift Route objects in the cluster.
// This service is specific to ACS Managed Services and provides methods to work on specific routes.
type RouteService struct {
	client ctrlClient.Client
}

// NewRouteService creates a new instance of RouteService.
func NewRouteService(client ctrlClient.Client) *RouteService {
	return &RouteService{client: client}
}

// FindReencryptRoute returns central reencrypt route or error if not found.
func (s *RouteService) FindReencryptRoute(ctx context.Context, namespace string) (*openshiftRouteV1.Route, error) {
	return s.findRoute(ctx, namespace, centralReencryptRouteName)
}

// FindPassthroughRoute returns central passthrough route or error if not found.
func (s *RouteService) FindPassthroughRoute(ctx context.Context, namespace string) (*openshiftRouteV1.Route, error) {
	return s.findRoute(ctx, namespace, centralPassthroughRouteName)
}

// FindReencryptIngress returns central reencrypt route ingress or error if not found.
func (s *RouteService) FindReencryptIngress(ctx context.Context, namespace string) (*openshiftRouteV1.RouteIngress, error) {
	return s.findFirstAdmittedIngress(ctx, namespace, centralReencryptRouteName)
}

// FindPassthroughIngress returns central passthrough route ingress or error if not found.
func (s *RouteService) FindPassthroughIngress(ctx context.Context, namespace string) (*openshiftRouteV1.RouteIngress, error) {
	return s.findFirstAdmittedIngress(ctx, namespace, centralPassthroughRouteName)
}

// findFirstAdmittedIngress returns first admitted ingress or error if not found
func (s *RouteService) findFirstAdmittedIngress(ctx context.Context, namespace string, routeName string) (*openshiftRouteV1.RouteIngress, error) {
	route, err := s.findRoute(ctx, namespace, routeName)
	if err != nil {
		return nil, fmt.Errorf("route not found")
	}
	for _, ingress := range route.Status.Ingress {
		if isAdmitted(ingress) {
			return &ingress, nil
		}
	}
	return nil, fmt.Errorf("unable to find admitted ingress. route: %s/%s", route.GetNamespace(), route.GetName())
}

func isAdmitted(ingress openshiftRouteV1.RouteIngress) bool {
	for _, condition := range ingress.Conditions {
		if condition.Type == openshiftRouteV1.RouteAdmitted {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// CreateReencryptRoute creates a new managed central reencrypt route.
func (s *RouteService) CreateReencryptRoute(ctx context.Context, remoteCentral private.ManagedCentral) error {
	centralTLSSecret := &v1.Secret{}
	namespace := remoteCentral.Metadata.Namespace
	err := s.client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: centralTLSSecretName}, centralTLSSecret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return ErrCentralTLSSecretNotFound
		}
		return errors.Wrapf(err, "get central TLS secret %s/%s", namespace, remoteCentral.Metadata.Name)
	}
	centralCA, ok := centralTLSSecret.Data["ca.pem"]
	if !ok {
		return errors.Errorf("could not find centrals ca certificate 'ca.pem' in secret/%s", centralTLSSecretName)
	}

	return s.createCentralRoute(ctx,
		centralReencryptRouteName,
		remoteCentral.Metadata.Namespace,
		remoteCentral.Spec.UiEndpoint.Host,
		&openshiftRouteV1.TLSConfig{
			Termination:              openshiftRouteV1.TLSTerminationReencrypt,
			Key:                      remoteCentral.Spec.UiEndpoint.Tls.Key,
			Certificate:              remoteCentral.Spec.UiEndpoint.Tls.Cert,
			DestinationCACertificate: string(centralCA),
		})
}

// CreatePassthroughRoute creates a new managed central passthrough route.
func (s *RouteService) CreatePassthroughRoute(ctx context.Context, remoteCentral private.ManagedCentral) error {
	return s.createCentralRoute(ctx,
		centralPassthroughRouteName,
		remoteCentral.Metadata.Namespace,
		remoteCentral.Spec.DataEndpoint.Host,
		&openshiftRouteV1.TLSConfig{
			Termination: openshiftRouteV1.TLSTerminationPassthrough,
		})
}

func (s *RouteService) createCentralRoute(ctx context.Context, name string, namespace string, host string, tls *openshiftRouteV1.TLSConfig) error {
	route := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{ManagedByLabelKey: ManagedByFleetshardValue},
		},
		Spec: openshiftRouteV1.RouteSpec{
			Host: host,
			Port: &openshiftRouteV1.RoutePort{
				TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "https"},
			},
			To: openshiftRouteV1.RouteTargetReference{
				Kind: "Service",
				Name: "central",
			},
			TLS: tls,
		},
	}

	if err := s.client.Create(ctx, route); err != nil {
		return fmt.Errorf("creating route %s/%s: %w", namespace, name, err)
	}
	return nil
}

func (s *RouteService) findRoute(ctx context.Context, namespace string, routeName string) (*openshiftRouteV1.Route, error) {
	route := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: namespace,
		},
	}
	err := s.client.Get(ctx, ctrlClient.ObjectKey{Namespace: route.GetNamespace(), Name: route.GetName()}, route)
	if err != nil {
		return route, fmt.Errorf("retrieving route %q from Kubernetes: %w", route.GetName(), err)
	}
	return route, nil
}
