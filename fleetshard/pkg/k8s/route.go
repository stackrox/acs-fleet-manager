package k8s

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/rox/pkg/errox"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralReencryptRouteName   = "managed-central-reencrypt"
	centralPassthroughRouteName = "managed-central-passthrough"

	routeAnnotationKeyRoot = "haproxy.router.openshift.io/"

	centralReencryptTimeoutAnnotationKey   = routeAnnotationKeyRoot + "timeout"
	centralReencryptTimeoutAnnotationValue = "10m"

	// Documentation for the route annotations to configure DDoS protection:
	// https://docs.openshift.com/container-platform/4.13/networking/routes/route-configuration.html#nw-route-specific-annotations_route-configuration
	rateLimitConnectionAnnotationKeyRoot = routeAnnotationKeyRoot + "rate-limit-connections"

	enableRateLimitAnnotationKey      = rateLimitConnectionAnnotationKeyRoot
	concurrentConnectionAnnotationKey = rateLimitConnectionAnnotationKeyRoot + ".concurrent-tcp"
	rateHTTPAnnotationKey             = rateLimitConnectionAnnotationKeyRoot + ".rate-http"
	rateTCPAnnotationKey              = rateLimitConnectionAnnotationKeyRoot + ".rate-tcp"
)

// RouteService is responsible for performing read and write operations on the OpenShift Route objects in the cluster.
// This service is specific to ACS Managed Services and provides methods to work on specific routes.
type RouteService struct {
	client ctrlClient.Client

	routeConfig *config.RouteConfig
}

// NewRouteService creates a new instance of RouteService.
func NewRouteService(client ctrlClient.Client, routeConfig *config.RouteConfig) *RouteService {
	return &RouteService{
		client:      client,
		routeConfig: routeConfig,
	}
}

func routeConfigAsAnnotationMap(routeConfig *config.RouteConfig) map[string]string {
	asAnnotationMap := make(map[string]string)
	asAnnotationMap[enableRateLimitAnnotationKey] = boolAsString(routeConfig.ThrottlingEnabled)
	asAnnotationMap[concurrentConnectionAnnotationKey] = intAsString(routeConfig.ConcurrentTCP)
	asAnnotationMap[rateHTTPAnnotationKey] = intAsString(routeConfig.RateHTTP)
	asAnnotationMap[rateTCPAnnotationKey] = intAsString(routeConfig.RateTCP)
	return asAnnotationMap
}

func boolAsString(boolValue bool) string {
	return fmt.Sprintf("%t", boolValue)
}

func intAsString(intValue int) string {
	return fmt.Sprintf("%d", intValue)
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

// FindAdmittedCloudServiceIngresses returns the list of admitted ingresses for all the Cloud Service routes
// for a given namespace or error if the list could not be retrieved
func (s *RouteService) FindAdmittedCloudServiceIngresses(ctx context.Context, namespace string) ([]openshiftRouteV1.RouteIngress, error) {
	routes := &openshiftRouteV1.RouteList{}
	if err := s.client.List(ctx, routes, ctrlClient.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("find admitted ingresses for namespace %s: %w", namespace, err)
	}
	var ingresses []openshiftRouteV1.RouteIngress
	for _, route := range routes.Items {
		if !isCloudServiceRoute(route) {
			continue
		}
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

func isCloudServiceRoute(route openshiftRouteV1.Route) bool {
	labels := route.ObjectMeta.Labels
	return labels[ManagedByLabelKey] == ManagedByFleetshardValue || labels["app.kubernetes.io/name"] == "tenant-resources"
}

type configureRouteFunc = func(context.Context, *openshiftRouteV1.Route, private.ManagedCentral) (*openshiftRouteV1.Route, error)

func (s *RouteService) hasExpectedTrafficLimitAnnotations(route *openshiftRouteV1.Route) bool {
	if route.ObjectMeta.Annotations == nil {
		return false
	}
	configAsAnnotations := routeConfigAsAnnotationMap(s.routeConfig)
	for k, v := range configAsAnnotations {
		if route.ObjectMeta.Annotations[k] != v {
			return false
		}
	}
	return true
}

func (s *RouteService) annotateRouteWithTrafficLimiters(route *openshiftRouteV1.Route) (*openshiftRouteV1.Route, error) {
	if route.ObjectMeta.Annotations == nil {
		route.ObjectMeta.Annotations = make(map[string]string)
	}
	configAsAnnotations := routeConfigAsAnnotationMap(s.routeConfig)
	for k, v := range configAsAnnotations {
		route.ObjectMeta.Annotations[k] = v
	}
	return route, nil
}

func (s *RouteService) configureReencryptRoute(ctx context.Context, route *openshiftRouteV1.Route, remoteCentral private.ManagedCentral) (*openshiftRouteV1.Route, error) {
	if route == nil {
		return nil, errox.InvalidArgs
	}
	annotatedRoute, annotationError := s.annotateRouteWithTrafficLimiters(route)
	if annotationError != nil {
		return nil, annotationError
	}
	annotatedRoute.ObjectMeta.Annotations[centralReencryptTimeoutAnnotationKey] = centralReencryptTimeoutAnnotationValue

	annotatedRoute.Spec.Host = remoteCentral.Spec.UiEndpoint.Host

	namespace := remoteCentral.Metadata.Namespace
	centralTLSSecret, retrievalErr := getSecret(ctx, s.client, CentralTLSSecretName, namespace)
	if retrievalErr != nil {
		wrappedErr := fmt.Errorf(
			"getting central-tls secret for tenant %s: %w",
			remoteCentral.Metadata.Name,
			retrievalErr,
		)
		return nil, wrappedErr
	}
	centralCA, ok := centralTLSSecret.Data["ca.pem"]
	if !ok {
		return nil, fmt.Errorf("could not find centrals ca certificate 'ca.pem' in secret/%s", CentralTLSSecretName)
	}

	if annotatedRoute.Spec.TLS == nil {
		annotatedRoute.Spec.TLS = &openshiftRouteV1.TLSConfig{}
	}
	annotatedRoute.Spec.TLS.Termination = openshiftRouteV1.TLSTerminationReencrypt
	annotatedRoute.Spec.TLS.Key = remoteCentral.Spec.UiEndpoint.Tls.Key
	annotatedRoute.Spec.TLS.Certificate = remoteCentral.Spec.UiEndpoint.Tls.Cert
	annotatedRoute.Spec.TLS.DestinationCACertificate = string(centralCA)

	return annotatedRoute, nil
}

// CreateReencryptRoute creates a new managed central reencrypt route.
func (s *RouteService) CreateReencryptRoute(ctx context.Context, remoteCentral private.ManagedCentral) error {
	return s.createCentralRoute(ctx,
		centralReencryptRouteName,
		remoteCentral.Metadata.Namespace,
		remoteCentral.Spec.UiEndpoint.Host,
		remoteCentral, s.configureReencryptRoute)
}

// UpdateReencryptRoute updates configuration of the given reencrytp route to match the TLS configuration of remoteCentral.
func (s *RouteService) UpdateReencryptRoute(ctx context.Context, route *openshiftRouteV1.Route, remoteCentral private.ManagedCentral) error {
	if route == nil {
		return errox.InvalidArgs
	}

	if s.reencryptConfigMatchesCentral(route, remoteCentral) &&
		s.hasExpectedTrafficLimitAnnotations(route) {
		return nil
	}

	updatedRoute, updateRouteErr := s.configureReencryptRoute(ctx, route.DeepCopy(), remoteCentral)
	if updateRouteErr != nil {
		return updateRouteErr
	}

	if err := s.client.Update(ctx, updatedRoute); err != nil {
		return fmt.Errorf("updating reencrypt route: %w", err)
	}

	return nil
}

func (s *RouteService) configurePassthroughRoute(_ context.Context, route *openshiftRouteV1.Route, remoteCentral private.ManagedCentral) (*openshiftRouteV1.Route, error) {
	annotatedRoute, annotationErr := s.annotateRouteWithTrafficLimiters(route)
	if annotationErr != nil {
		return nil, annotationErr
	}
	annotatedRoute.Spec.Host = remoteCentral.Spec.DataEndpoint.Host
	if annotatedRoute.Spec.TLS == nil {
		annotatedRoute.Spec.TLS = &openshiftRouteV1.TLSConfig{}
	}
	annotatedRoute.Spec.TLS.Termination = openshiftRouteV1.TLSTerminationPassthrough

	return annotatedRoute, nil
}

// UpdatePassthroughRoute updates configuration of the given passthrough route to match remoteCentral.
func (s *RouteService) UpdatePassthroughRoute(ctx context.Context, route *openshiftRouteV1.Route, remoteCentral private.ManagedCentral) error {
	if route == nil {
		return errox.InvalidArgs
	}

	if route.Spec.Host == remoteCentral.Spec.DataEndpoint.Host &&
		s.hasExpectedTrafficLimitAnnotations(route) {
		return nil
	}

	updatedRoute, updateRouteErr := s.configurePassthroughRoute(ctx, route.DeepCopy(), remoteCentral)
	if updateRouteErr != nil {
		return updateRouteErr
	}

	if err := s.client.Update(ctx, updatedRoute); err != nil {
		return fmt.Errorf("updating passthrough route: %w", err)
	}

	return nil
}

// CreatePassthroughRoute creates a new managed central passthrough route.
func (s *RouteService) CreatePassthroughRoute(ctx context.Context, remoteCentral private.ManagedCentral) error {
	return s.createCentralRoute(ctx,
		centralPassthroughRouteName,
		remoteCentral.Metadata.Namespace,
		remoteCentral.Spec.DataEndpoint.Host,
		remoteCentral, s.configurePassthroughRoute)
}

func (s *RouteService) createCentralRoute(
	ctx context.Context,
	name, namespace, host string,
	remoteCentral private.ManagedCentral,
	configureRoute configureRouteFunc,
) error {
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
		},
	}
	configuredRoute, configureErr := configureRoute(ctx, route, remoteCentral)
	if configureErr != nil {
		return configureErr
	}

	if err := s.client.Create(ctx, configuredRoute); err != nil {
		return fmt.Errorf("creating route %s/%s: %w", namespace, name, err)
	}
	return nil
}

// DeleteReencryptRoute deletes central reencrypt route for a given namespace
func (s *RouteService) DeleteReencryptRoute(ctx context.Context, namespace string) error {
	return s.deleteRoute(ctx, centralReencryptRouteName, namespace)
}

// DeletePassthroughRoute deletes central passthrough route for a given namespace
func (s *RouteService) DeletePassthroughRoute(ctx context.Context, namespace string) error {
	return s.deleteRoute(ctx, centralPassthroughRouteName, namespace)
}

func (s *RouteService) deleteRoute(ctx context.Context, name string, namespace string) error {
	route := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{ManagedByLabelKey: ManagedByFleetshardValue},
		},
	}
	if err := s.client.Delete(ctx, route); err != nil {
		return fmt.Errorf("deleting route %s/%s: %w", namespace, name, err)
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

func (s *RouteService) reencryptConfigMatchesCentral(route *openshiftRouteV1.Route, remoteCentral private.ManagedCentral) bool {
	return route.Spec.TLS.Certificate == remoteCentral.Spec.UiEndpoint.Tls.Cert &&
		route.Spec.TLS.Key == remoteCentral.Spec.UiEndpoint.Tls.Key &&
		route.Spec.Host == remoteCentral.Spec.UiEndpoint.Host
}
