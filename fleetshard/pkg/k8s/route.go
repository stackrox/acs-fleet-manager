package k8s

import (
	"context"
	"fmt"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralReencryptRouteName = "central-reencrypt"
	centralMtlsRouteName      = "central-mtls"
	centralTLSSecretName      = "central-tls"
)

type RouteService struct {
	client ctrlClient.Client
}

func NewRouteService(client ctrlClient.Client) *RouteService {
	return &RouteService{client: client}
}

func (s *RouteService) FindReencryptRoute(ctx context.Context, namespace string) (*openshiftRouteV1.Route, error) {
	return s.findRoute(ctx, namespace, centralReencryptRouteName)
}

func (s *RouteService) FindReencryptCanonicalHostname(ctx context.Context, namespace string) (string, error) {
	return s.findCanonicalHostname(ctx, namespace, centralReencryptRouteName)
}

func (s *RouteService) FindMtlsCanonicalHostname(ctx context.Context, namespace string) (string, error) {
	return s.findCanonicalHostname(ctx, namespace, centralMtlsRouteName)
}

func (s *RouteService) findCanonicalHostname(ctx context.Context, namespace string, routeName string) (string, error) {
	route, err := s.findRoute(ctx, namespace, routeName)
	if err != nil {
		return "", err
	}
	for _, ingress := range route.Status.Ingress {
		if isAdmitted(ingress) {
			return ingress.RouterCanonicalHostname, nil
		}
	}
	return "", fmt.Errorf("route canonical hostname is not found. route: %s/%s", namespace, routeName)
}

func isAdmitted(ingress openshiftRouteV1.RouteIngress) bool {
	for _, condition := range ingress.Conditions {
		if condition.Type == openshiftRouteV1.RouteAdmitted {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func newReencryptRoute(namespace string) *openshiftRouteV1.Route {
	return &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralReencryptRouteName,
			Namespace: namespace,
			Labels:    map[string]string{ManagedByLabelKey: ManagedByLabelValue},
		},
	}
}

func (s *RouteService) CreateReencryptRoute(ctx context.Context, remoteCentral private.ManagedCentral) error {
	centralTLSSecret := &v1.Secret{}
	namespace := remoteCentral.Metadata.Namespace
	err := s.client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: centralTLSSecretName}, centralTLSSecret)
	if err != nil {
		return errors.Wrapf(err, "get central TLS secret %s/%s", namespace, remoteCentral.Metadata.Name)
	}
	centralCA, ok := centralTLSSecret.Data["ca.pem"]
	if !ok {
		return errors.Errorf("could not find centrals ca certificate 'ca.pem' in secret/%s", centralTLSSecretName)
	}
	route := newReencryptRoute(namespace)
	route.Spec = openshiftRouteV1.RouteSpec{
		Host: remoteCentral.Spec.Endpoint.Host,
		Port: &openshiftRouteV1.RoutePort{
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "https"},
		},
		To: openshiftRouteV1.RouteTargetReference{
			Kind: "Service",
			Name: "central",
		},
		TLS: &openshiftRouteV1.TLSConfig{
			Termination:              openshiftRouteV1.TLSTerminationReencrypt,
			Key:                      remoteCentral.Spec.Endpoint.Tls.Key,
			Certificate:              remoteCentral.Spec.Endpoint.Tls.Cert,
			DestinationCACertificate: string(centralCA),
		},
	}
	return s.client.Create(ctx, route)
}

func (s *RouteService) findRoute(ctx context.Context, namespace string, routeName string) (*openshiftRouteV1.Route, error) {
	route := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: namespace,
		},
	}
	err := s.client.Get(ctx, ctrlClient.ObjectKey{Namespace: route.GetNamespace(), Name: route.GetName()}, route)
	return route, err
}
