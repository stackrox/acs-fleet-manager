package testutils

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-multierror"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sTesting "k8s.io/client-go/testing"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	centralsGVR = schema.GroupVersionResource{
		Group:    "platform.stackrox.io",
		Version:  "v1alpha1",
		Resource: "centrals",
	}
	// pragma: allowlist nextline secret
	secretsGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "secrets",
	}
	routesGVR = schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}
)

// CentralCA ...
const (
	// CentralCA ...
	CentralCA     = "test CA"
	clusterDomain = "test.local"
)

type reconcileTracker struct {
	k8sTesting.ObjectTracker
}

// NewFakeClientBuilder returns a new fake client builder with registered custom resources
func NewFakeClientBuilder(t *testing.T, objects ...ctrlClient.Object) *fake.ClientBuilder {
	scheme := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, openshiftRouteV1.Install(scheme))
	require.NoError(t, openshiftOperatorV1.Install(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjectTracker(newReconcileTracker(scheme)).
		WithObjects(objects...)
}

func newReconcileTracker(scheme *runtime.Scheme) k8sTesting.ObjectTracker {
	return reconcileTracker{ObjectTracker: k8sTesting.NewObjectTracker(scheme, clientgoscheme.Codecs.UniversalDecoder())}
}

// Create ...
func (t reconcileTracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	if gvr == routesGVR {
		route := obj.(*openshiftRouteV1.Route)
		route.Status = admittedStatus(route.Spec.Host)
	}
	if err := t.ObjectTracker.Create(gvr, obj, ns); err != nil {
		return fmt.Errorf("adding GVR %q to reconcile tracker: %w", gvr, err)
	}
	if gvr == centralsGVR {
		var multiErr *multierror.Error
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(secretsGVR, newCentralTLSSecret(ns), ns))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(routesGVR, newCentralRoute(ns), ns))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(routesGVR, newCentralMtlsRoute(ns), ns))
		err := multiErr.ErrorOrNil()
		if err != nil {
			return fmt.Errorf("creating group version resource: %w", err)
		}
	}
	return nil
}

func newCentralTLSSecret(ns string) *coreV1.Secret {
	return &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"ca.pem": []byte(CentralCA),
		},
	}
}

func newCentralRoute(ns string) *openshiftRouteV1.Route {
	host := fmt.Sprintf("central-%s.%s", ns, clusterDomain)
	return &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "stackrox",
				"app.kubernetes.io/component": "central",
			},
		},
		Spec: openshiftRouteV1.RouteSpec{
			Host: host,
		},
		Status: admittedStatus(host),
	}
}

func newCentralMtlsRoute(ns string) *openshiftRouteV1.Route {
	host := fmt.Sprintf("central.%s", ns)
	return &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-mtls",
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "stackrox",
				"app.kubernetes.io/component": "central",
			},
		},
		Spec: openshiftRouteV1.RouteSpec{
			Host: host,
		},
		Status: admittedStatus(host),
	}
}

func admittedStatus(host string) openshiftRouteV1.RouteStatus {
	return openshiftRouteV1.RouteStatus{
		Ingress: []openshiftRouteV1.RouteIngress{
			{
				Conditions: []openshiftRouteV1.RouteIngressCondition{
					{
						Type:   openshiftRouteV1.RouteAdmitted,
						Status: coreV1.ConditionTrue,
					},
				},
				Host:                    host,
				RouterCanonicalHostname: "router-default.apps.test.local",
			},
		},
	}
}
