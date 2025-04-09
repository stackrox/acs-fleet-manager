package k8s

import (
	"context"
	"fmt"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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
