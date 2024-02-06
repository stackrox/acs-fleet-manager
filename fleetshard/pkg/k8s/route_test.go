package k8s

import (
	"context"
	"fmt"
	"testing"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	enabledLabelValue  = "true"
	disabledLabelValue = "false"

	baseConcurrentTCPStrVal = "32"
	baseRateHTTPStrVal      = "128"
	baseRateTCPStrVal       = "16"

	updatedConcurrentTCPStrVal = "16"
	updatedRateHTTPStrVal      = "512"
	updatedRateTCPStrVal       = "8"

	testNamespace           = "test-namespace"
	testTargetHost          = "target-host"
	testTargetReEncryptHost = "target-re-encrypt-host"
	testHTTPSTarget         = "https"
	testServiceRouteToKind  = "Service"
	testCentralService      = "central"
	testTLSCert             = "This is a dummy certificate"
	testTLSKey              = "This is a dummy TLS Key"
)

var (
	baseRouteParameters = &config.RouteConfig{
		ThrottlingEnabled: true,
		ConcurrentTCP:     32,
		RateHTTP:          128,
		RateTCP:           16,
	}

	updatedRouteParameters = &config.RouteConfig{
		ThrottlingEnabled: false,
		ConcurrentTCP:     16,
		RateHTTP:          512,
		RateTCP:           8,
	}
)

var (
	passThroughRouteSpec = openshiftRouteV1.RouteSpec{
		Host: testTargetHost,
		Port: &openshiftRouteV1.RoutePort{
			TargetPort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: testHTTPSTarget,
			},
		},
		To: openshiftRouteV1.RouteTargetReference{
			Kind: testServiceRouteToKind,
			Name: testCentralService,
		},
		TLS: &openshiftRouteV1.TLSConfig{
			Termination: openshiftRouteV1.TLSTerminationPassthrough,
		},
	}

	expectedCreatedPassThroughRoute = &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralPassthroughRouteName,
			Namespace: testNamespace,
			Labels: map[string]string{
				ManagedByLabelKey: ManagedByFleetshardValue,
			},
			Annotations: map[string]string{
				enableRateLimitAnnotationKey:      enabledLabelValue,
				concurrentConnectionAnnotationKey: baseConcurrentTCPStrVal,
				rateHTTPAnnotationKey:             baseRateHTTPStrVal,
				rateTCPAnnotationKey:              baseRateTCPStrVal,
			},
		},
		Spec: passThroughRouteSpec,
	}

	expectedUpdatedPassThroughRoute = &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralPassthroughRouteName,
			Namespace: testNamespace,
			Labels: map[string]string{
				ManagedByLabelKey: ManagedByFleetshardValue,
			},
			Annotations: map[string]string{
				enableRateLimitAnnotationKey:      disabledLabelValue,
				concurrentConnectionAnnotationKey: updatedConcurrentTCPStrVal,
				rateHTTPAnnotationKey:             updatedRateHTTPStrVal,
				rateTCPAnnotationKey:              updatedRateTCPStrVal,
			},
		},
		Spec: passThroughRouteSpec,
	}
)

var (
	passThroughRouteExtractors = map[string]routeFieldExtractor{
		`name`:                          extractRouteObjectMetaName,
		`namespace`:                     extractRouteObjectMetaNamespace,
		`"managed-by" label`:            extractRouteManagedByLabel,
		`spec host`:                     extractRouteSpecHost,
		`spec target port type`:         extractRouteSpecTargetPortType,
		`spec target port string value`: extractRouteSpecTargetPortStrVal,
		`spec "to" kind`:                extractRouteSpecToKind,
		`spec "to" name`:                extractRouteSpecToName,
		`spec TLS termination`:          extractRouteSpecTLSTermination,
		// Rate limit annotations
		`"rate-limit-connections" annotation`:                extractRouteEnableRateLimitAnnotation,
		`"rate-limit-connections.concurrent-tcp" annotation`: extractRouteConcurrentConnectionsAnnotation,
		`"rate-limit-connections.rate-http" annotation`:      extractRouteRateHTTPAnnotation,
		`"rate-limit-connections.rate-tcp" annotation`:       extractRouteRateTCPAnnotation,
	}
)

func TestPassThroughRouteLifecycle(t *testing.T) {
	client := testutils.NewFakeClientBuilder(t).Build()
	baseRouteService := NewRouteService(client, baseRouteParameters)
	updateRouteService := NewRouteService(client, updatedRouteParameters)

	// Test configuration
	ctx := context.Background()

	remoteCentral := private.ManagedCentral{
		Id:   uuid.NewV4().String(),
		Kind: "central",
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      "test central 1",
			Namespace: testNamespace,
		},
		Spec: private.ManagedCentralAllOfSpec{
			DataEndpoint: private.ManagedCentralAllOfSpecDataEndpoint{
				Host: testTargetHost,
			},
		},
	}

	// Ensure FindPassThroughRoute does not find anything when the route was not created yet.
	unConfiguredPassThroughRoute := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      centralPassthroughRouteName,
		},
	}

	missingRoute, missingLookupErr := baseRouteService.FindPassthroughRoute(ctx, testNamespace)
	assert.Equal(t, missingRoute, unConfiguredPassThroughRoute)
	assert.True(t, apiErrors.IsNotFound(missingLookupErr))

	missingUpdatedRoute, missingUpdatedLookupErr := updateRouteService.FindPassthroughRoute(ctx, testNamespace)
	assert.Equal(t, missingUpdatedRoute, unConfiguredPassThroughRoute)
	assert.True(t, apiErrors.IsNotFound(missingUpdatedLookupErr))

	// Validate behaviour of update on non-existing route and of create
	updateNonExistingErr := updateRouteService.UpdatePassthroughRoute(ctx, missingUpdatedRoute, remoteCentral)
	assert.True(t, apiErrors.IsNotFound(updateNonExistingErr))

	createErr := baseRouteService.CreatePassthroughRoute(ctx, remoteCentral)
	require.NoError(t, createErr)

	// Validate parameters of the created object
	createdRouteKey := ctrlClient.ObjectKey{
		Namespace: testNamespace,
		Name:      centralPassthroughRouteName,
	}
	createdRoute := &openshiftRouteV1.Route{}
	require.NoError(t, client.Get(ctx, createdRouteKey, createdRoute))
	compareRoutes(t, createdRoute, expectedCreatedPassThroughRoute, passThroughRouteExtractors)

	// Check base and update services work on the same route
	postCreateRoute, postCreateLookupErr := baseRouteService.FindPassthroughRoute(ctx, testNamespace)
	assert.NoError(t, postCreateLookupErr)
	assert.Equal(t, createdRoute, postCreateRoute)
	preUpdateRoute, preUpdateLookupErr := updateRouteService.FindPassthroughRoute(ctx, testNamespace)
	assert.NoError(t, preUpdateLookupErr)
	assert.Equal(t, createdRoute, preUpdateRoute)

	updateErr := updateRouteService.UpdatePassthroughRoute(ctx, preUpdateRoute, remoteCentral)
	assert.NoError(t, updateErr)

	// Validate update only changed the fields that differ between baseRouteParams and updateRouteParams
	updatedRoute := &openshiftRouteV1.Route{}
	require.NoError(t, client.Get(ctx, createdRouteKey, updatedRoute))
	compareRoutes(t, updatedRoute, expectedUpdatedPassThroughRoute, passThroughRouteExtractors)
}

var (
	centralTLSSecret = &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CentralTLSSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"ca.pem": []byte("This is a dummy non-pem payload for CA"),
		},
	}

	reEncryptRouteSpec = openshiftRouteV1.RouteSpec{
		Host: testTargetReEncryptHost,
		Port: &openshiftRouteV1.RoutePort{
			TargetPort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: testHTTPSTarget,
			},
		},
		To: openshiftRouteV1.RouteTargetReference{
			Kind: testServiceRouteToKind,
			Name: testCentralService,
		},
		TLS: &openshiftRouteV1.TLSConfig{
			Termination:              openshiftRouteV1.TLSTerminationReencrypt,
			Certificate:              testTLSCert,
			Key:                      testTLSKey,
			DestinationCACertificate: string(centralTLSSecret.Data["ca.pem"]),
		},
	}

	expectedCreatedReEncryptRoute = &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralReencryptRouteName,
			Namespace: testNamespace,
			Labels: map[string]string{
				ManagedByLabelKey: ManagedByFleetshardValue,
			},
			Annotations: map[string]string{
				enableRateLimitAnnotationKey:         enabledLabelValue,
				concurrentConnectionAnnotationKey:    baseConcurrentTCPStrVal,
				rateHTTPAnnotationKey:                baseRateHTTPStrVal,
				rateTCPAnnotationKey:                 baseRateTCPStrVal,
				centralReencryptTimeoutAnnotationKey: centralReencryptTimeoutAnnotationValue,
			},
		},
		Spec: reEncryptRouteSpec,
	}

	expectedUpdatedReEncryptRoute = &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralReencryptRouteName,
			Namespace: testNamespace,
			Labels: map[string]string{
				ManagedByLabelKey: ManagedByFleetshardValue,
			},
			Annotations: map[string]string{
				enableRateLimitAnnotationKey:         disabledLabelValue,
				concurrentConnectionAnnotationKey:    updatedConcurrentTCPStrVal,
				rateHTTPAnnotationKey:                updatedRateHTTPStrVal,
				rateTCPAnnotationKey:                 updatedRateTCPStrVal,
				centralReencryptTimeoutAnnotationKey: centralReencryptTimeoutAnnotationValue,
			},
		},
		Spec: reEncryptRouteSpec,
	}

	reEncryptRouteExtractors = map[string]routeFieldExtractor{
		`name`:                                extractRouteObjectMetaName,
		`namespace`:                           extractRouteObjectMetaNamespace,
		`"managed-by" label`:                  extractRouteManagedByLabel,
		`"timeout" annotation`:                extractRouteReEncryptTimeoutAnnotation,
		`spec host`:                           extractRouteSpecHost,
		`spec target port type`:               extractRouteSpecTargetPortType,
		`spec target port string value`:       extractRouteSpecTargetPortStrVal,
		`spec "to" kind`:                      extractRouteSpecToKind,
		`spec "to" name`:                      extractRouteSpecToName,
		`spec TLS termination`:                extractRouteSpecTLSTermination,
		`spec TLS destination CA certificate`: extractRouteSpecTLSDestinationCACertificate,
		`spec TLS key`:                        extractRouteSpecTLSKey,
		`spec TLS certificate`:                extractRouteSpecTLSCertificate,
		// Rate limit annotations
		`"rate-limit-connections" annotation`:                extractRouteEnableRateLimitAnnotation,
		`"rate-limit-connections.concurrent-tcp" annotation`: extractRouteConcurrentConnectionsAnnotation,
		`"rate-limit-connections.rate-http" annotation`:      extractRouteRateHTTPAnnotation,
		`"rate-limit-connections.rate-tcp" annotation`:       extractRouteRateTCPAnnotation,
	}
)

func TestReEncryptRouteLifecycle(t *testing.T) {
	const testNamespace = "test-namespace"
	client := testutils.NewFakeClientBuilder(t).Build()
	baseRouteService := NewRouteService(client, baseRouteParameters)
	updateRouteService := NewRouteService(client, updatedRouteParameters)

	// Test configuration
	ctx := context.Background()

	remoteCentral := private.ManagedCentral{
		Id:   uuid.NewV4().String(),
		Kind: "central",
		Metadata: private.ManagedCentralAllOfMetadata{
			Name:      "test central 1",
			Namespace: testNamespace,
		},
		Spec: private.ManagedCentralAllOfSpec{
			UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
				Host: testTargetReEncryptHost,
				Tls: private.ManagedCentralAllOfSpecUiEndpointTls{
					Cert: testTLSCert,
					Key:  testTLSKey,
				},
			},
		},
	}

	require.NoError(t, client.Create(ctx, centralTLSSecret))

	// Ensure FindReEncryptRoute does not find anything when the route was not created yet.
	unConfiguredReEncryptRoute := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      centralReencryptRouteName,
		},
	}

	missingRoute, missingLookupErr := baseRouteService.FindReencryptRoute(ctx, testNamespace)
	assert.Equal(t, missingRoute, unConfiguredReEncryptRoute)
	assert.True(t, apiErrors.IsNotFound(missingLookupErr))

	missingUpdatedRoute, missingUpdatedLookupErr := updateRouteService.FindReencryptRoute(ctx, testNamespace)
	assert.Equal(t, missingUpdatedRoute, unConfiguredReEncryptRoute)
	assert.True(t, apiErrors.IsNotFound(missingUpdatedLookupErr))

	createErr := baseRouteService.CreateReencryptRoute(ctx, remoteCentral)
	require.NoError(t, createErr)

	// Validate parameters of the created object
	createdRouteKey := ctrlClient.ObjectKey{
		Namespace: testNamespace,
		Name:      centralReencryptRouteName,
	}
	createdRoute := &openshiftRouteV1.Route{}
	require.NoError(t, client.Get(ctx, createdRouteKey, createdRoute))
	compareRoutes(t, createdRoute, expectedCreatedReEncryptRoute, reEncryptRouteExtractors)

	// Check base and update services work on the same route
	postCreateRoute, postCreateLookupErr := baseRouteService.FindReencryptRoute(ctx, testNamespace)
	assert.NoError(t, postCreateLookupErr)
	assert.Equal(t, createdRoute, postCreateRoute)
	preUpdateRoute, preUpdateLookupErr := updateRouteService.FindReencryptRoute(ctx, testNamespace)
	assert.NoError(t, preUpdateLookupErr)
	assert.Equal(t, createdRoute, preUpdateRoute)

	require.NotNil(t, preUpdateRoute.Spec.TLS)

	updateErr := updateRouteService.UpdateReencryptRoute(ctx, preUpdateRoute, remoteCentral)
	assert.NoError(t, updateErr)

	// Validate update only changed the fields that differ between baseRouteParams and updateRouteParams
	updatedRoute := &openshiftRouteV1.Route{}
	require.NoError(t, client.Get(ctx, createdRouteKey, updatedRoute))
	compareRoutes(t, updatedRoute, expectedUpdatedReEncryptRoute, reEncryptRouteExtractors)
}

// region Validation helpers

type routeFieldExtractor func(route *openshiftRouteV1.Route) string

func compareRoutes(
	t *testing.T,
	expectedRoute *openshiftRouteV1.Route,
	testedRoute *openshiftRouteV1.Route,
	extractors map[string]routeFieldExtractor,
) {
	for field, extract := range extractors {
		expectedVal := extract(expectedRoute)
		testedVal := extract(testedRoute)
		description := fmt.Sprintf("Route %s comparison - expected %q - got %q", field, expectedVal, testedVal)
		assert.Equal(t, expectedVal, testedVal, description)
	}
}

func extractRouteObjectMetaName(route *openshiftRouteV1.Route) string {
	return route.ObjectMeta.Name
}

func extractRouteObjectMetaNamespace(route *openshiftRouteV1.Route) string {
	return route.ObjectMeta.Namespace
}

func extractRouteManagedByLabel(route *openshiftRouteV1.Route) string {
	if route.ObjectMeta.Labels == nil {
		return ""
	}
	return route.ObjectMeta.Labels[ManagedByLabelKey]
}

func extractRouteAnnotation(route *openshiftRouteV1.Route, key string) string {
	if route.ObjectMeta.Annotations == nil {
		return ""
	}
	return route.ObjectMeta.Annotations[key]
}

func extractRouteEnableRateLimitAnnotation(route *openshiftRouteV1.Route) string {
	return extractRouteAnnotation(route, enableRateLimitAnnotationKey)
}

func extractRouteConcurrentConnectionsAnnotation(route *openshiftRouteV1.Route) string {
	return extractRouteAnnotation(route, concurrentConnectionAnnotationKey)
}

func extractRouteRateHTTPAnnotation(route *openshiftRouteV1.Route) string {
	return extractRouteAnnotation(route, rateHTTPAnnotationKey)
}

func extractRouteRateTCPAnnotation(route *openshiftRouteV1.Route) string {
	return extractRouteAnnotation(route, rateTCPAnnotationKey)
}

func extractRouteReEncryptTimeoutAnnotation(route *openshiftRouteV1.Route) string {
	return extractRouteAnnotation(route, centralReencryptTimeoutAnnotationKey)
}

func extractRouteSpecHost(route *openshiftRouteV1.Route) string {
	return route.Spec.Host
}

func extractRouteSpecTargetPortType(route *openshiftRouteV1.Route) string {
	if route.Spec.Port == nil {
		return ""
	}
	return fmt.Sprintf("%d", route.Spec.Port.TargetPort.Type)
}

func extractRouteSpecTargetPortStrVal(route *openshiftRouteV1.Route) string {
	if route.Spec.Port == nil {
		return ""
	}
	return route.Spec.Port.TargetPort.StrVal
}

func extractRouteSpecToKind(route *openshiftRouteV1.Route) string {
	return route.Spec.To.Kind
}

func extractRouteSpecToName(route *openshiftRouteV1.Route) string {
	return route.Spec.To.Name
}

func extractRouteSpecTLSTermination(route *openshiftRouteV1.Route) string {
	if route.Spec.TLS == nil {
		return ""
	}
	return string(route.Spec.TLS.Termination)
}

func extractRouteSpecTLSDestinationCACertificate(route *openshiftRouteV1.Route) string {
	if route.Spec.TLS == nil {
		return ""
	}
	return route.Spec.TLS.DestinationCACertificate
}

func extractRouteSpecTLSKey(route *openshiftRouteV1.Route) string {
	if route.Spec.TLS == nil {
		return ""
	}
	return route.Spec.TLS.Key
}

func extractRouteSpecTLSCertificate(route *openshiftRouteV1.Route) string {
	if route.Spec.TLS == nil {
		return ""
	}
	return route.Spec.TLS.Certificate
}

// endregion Validation helpers
