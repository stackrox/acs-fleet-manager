package k8s

import (
	"context"
	"fmt"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralReencryptRouteName   = "managed-central-reencrypt"
	centralPassthroughRouteName = "managed-central-passthrough"
)

// RouteService is responsible for performing read and write operations on the OpenShift Route objects in the cluster.
// This service is specific to ACS Managed Services and provides methods to work on specific routes.
type RouteService struct {
	client ctrlClient.Client
}

// NewRouteService creates a new instance of RouteService.
func NewRouteService(client ctrlClient.Client) *RouteService {
	return &RouteService{
		client: client,
	}
}

// FindReencryptRoute returns central reencrypt route or error if not found.
func (s *RouteService) FindReencryptRoute(ctx context.Context, namespace string) (*openshiftRouteV1.Route, error) {
	return s.findRoute(ctx, namespace, centralReencryptRouteName)
}

// FindPassthroughRoute returns central passthrough route or error if not found.
func (s *RouteService) FindPassthroughRoute(ctx context.Context, namespace string) (*openshiftRouteV1.Route, error) {
	return s.findRoute(ctx, namespace, centralPassthroughRouteName)
}

// FindReencryptIngress returns central reencrypt route ingress or nil if not found.
// The error is returned when failed to get the route.
func (s *RouteService) FindReencryptIngress(ctx context.Context, namespace string) (*openshiftRouteV1.RouteIngress, error) {
	route, err := s.FindReencryptRoute(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return findFirstAdmittedIngress(*route), nil
}

// FindAdmittedIngresses returns the list of admitted ingresses for a given namespace or error if the list could not be retrieved
func (s *RouteService) FindAdmittedIngresses(ctx context.Context, namespace string) ([]openshiftRouteV1.RouteIngress, error) {
	routes := &openshiftRouteV1.RouteList{}
	if err := s.client.List(ctx, routes, ctrlClient.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("find admitted ingresses for namespace %s: %w", namespace, err)
	}
	var ingresses []openshiftRouteV1.RouteIngress
	for _, route := range routes.Items {
		admittedIngress := findFirstAdmittedIngress(route)
		if admittedIngress != nil {
			ingresses = append(ingresses, *admittedIngress)
		}
	}
	return ingresses, nil
}

// findFirstAdmittedIngress returns first admitted ingress or nil if not found
func findFirstAdmittedIngress(route openshiftRouteV1.Route) *openshiftRouteV1.RouteIngress {
	for _, ingress := range route.Status.Ingress {
		if isAdmitted(ingress) {
			return &ingress
		}
	}
	return nil
}

func isAdmitted(ingress openshiftRouteV1.RouteIngress) bool {
	for _, condition := range ingress.Conditions {
		if condition.Type == openshiftRouteV1.RouteAdmitted {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
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
