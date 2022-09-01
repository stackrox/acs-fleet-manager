package testutils

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-multierror"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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
	deploymentGVR = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
)

const (
	// CentralCA certificate authority which is used by central and included with the stackrox distribution.
	CentralCA     = "test CA"
	clusterDomain = "test.local"
)

var centralLabels = map[string]string{
	"app.kubernetes.io/name":      "stackrox",
	"app.kubernetes.io/component": "central",
}

// ReconcileTracker keeps track of objects. It is intended to be used to
// fake calls to a server by returning objects based on their kind,
// namespace and name. This is fleetshard specific implementation of k8sTesting.ObjectTracker
type ReconcileTracker struct {
	k8sTesting.ObjectTracker
	routeErrors     map[string]error
	routeConditions map[string]*openshiftRouteV1.RouteIngressCondition
	skipRoute       map[string]bool
}

// NewFakeClientBuilder returns a new fake client builder with registered custom resources
func NewFakeClientBuilder(t *testing.T, objects ...ctrlClient.Object) *fake.ClientBuilder {
	scheme := NewScheme(t)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjectTracker(NewReconcileTracker(scheme)).
		WithObjects(objects...)
}

// NewFakeClientWithTracker returns a new fake client and a ReconcileTracker to mock k8s responses
func NewFakeClientWithTracker(t *testing.T) (ctrlClient.WithWatch, *ReconcileTracker) {
	scheme := NewScheme(t)

	tracker := NewReconcileTracker(scheme)
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjectTracker(tracker).
		Build()
	return client, tracker
}

// NewScheme returns a new scheme instance used for fleetshard testing
func NewScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()

	require.NoError(t, coreV1.AddToScheme(scheme))
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, openshiftRouteV1.Install(scheme))
	require.NoError(t, openshiftOperatorV1.Install(scheme))

	return scheme
}

// NewReconcileTracker creates a new instance of ReconcileTracker
func NewReconcileTracker(scheme *runtime.Scheme) *ReconcileTracker {
	return &ReconcileTracker{
		ObjectTracker:   k8sTesting.NewObjectTracker(scheme, clientgoscheme.Codecs.UniversalDecoder()),
		routeErrors:     map[string]error{},
		routeConditions: map[string]*openshiftRouteV1.RouteIngressCondition{},
		skipRoute:       map[string]bool{},
	}
}

// AddRouteError add a new error on a given route creation
func (t *ReconcileTracker) AddRouteError(routeName string, err error) {
	t.routeErrors[routeName] = err
}

// SetRouteAdmitted add a rule to set RouteIngressCondition for a given route
func (t *ReconcileTracker) SetRouteAdmitted(routeName string, admitted bool) {
	condition := &openshiftRouteV1.RouteIngressCondition{
		Type: openshiftRouteV1.RouteAdmitted,
	}
	if admitted {
		condition.Status = coreV1.ConditionTrue
	} else {
		condition.Status = coreV1.ConditionFalse
	}
	t.routeConditions[routeName] = condition
}

// SetSkipRoute do not create route with a given name when flag is true
func (t *ReconcileTracker) SetSkipRoute(routeName string, skip bool) {
	t.skipRoute[routeName] = skip
}

// Create adds an object to the tracker in the specified namespace.
func (t *ReconcileTracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	if gvr == routesGVR {
		route := obj.(*openshiftRouteV1.Route)
		route.Status = t.admittedStatus(route.Name, route.Spec.Host)
		return t.createRoute(route)
	}
	if err := t.ObjectTracker.Create(gvr, obj, ns); err != nil {
		return fmt.Errorf("adding GVR %q to reconcile tracker: %w", gvr, err)
	}
	if gvr == centralsGVR {
		var multiErr *multierror.Error
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(secretsGVR, newCentralTLSSecret(ns), ns))
		multiErr = multierror.Append(multiErr, t.createRoute(t.newCentralRoute(ns)))
		multiErr = multierror.Append(multiErr, t.createRoute(t.newCentralMtlsRoute(ns)))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(deploymentGVR, NewCentralDeployment(ns), ns))
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

func (t *ReconcileTracker) createRoute(route *openshiftRouteV1.Route) error {
	name := route.GetName()
	if err := t.routeErrors[name]; err != nil {
		return err
	}
	if t.skipRoute[name] {
		return nil
	}
	err := t.ObjectTracker.Create(routesGVR, route, route.GetNamespace())
	return errors.Wrapf(err, "create route")
}

func (t *ReconcileTracker) newCentralRoute(ns string) *openshiftRouteV1.Route {
	host := fmt.Sprintf("central-%s.%s", ns, clusterDomain)
	name := "central"
	return &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    centralLabels,
		},
		Spec: openshiftRouteV1.RouteSpec{
			Host: host,
		},
		Status: t.admittedStatus(name, host),
	}
}

func (t *ReconcileTracker) newCentralMtlsRoute(ns string) *openshiftRouteV1.Route {
	host := fmt.Sprintf("central.%s", ns)
	name := "central-mtls"
	return &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    centralLabels,
		},
		Spec: openshiftRouteV1.RouteSpec{
			Host: host,
		},
		Status: t.admittedStatus(name, host),
	}
}

func (t *ReconcileTracker) admittedStatus(routeName string, host string) openshiftRouteV1.RouteStatus {
	routeCondition := t.routeConditions[routeName]
	if routeCondition == nil {
		routeCondition = &openshiftRouteV1.RouteIngressCondition{
			Type:   openshiftRouteV1.RouteAdmitted,
			Status: coreV1.ConditionTrue,
		}
	}

	return openshiftRouteV1.RouteStatus{
		Ingress: []openshiftRouteV1.RouteIngress{
			{
				Conditions:              []openshiftRouteV1.RouteIngressCondition{*routeCondition},
				Host:                    host,
				RouterCanonicalHostname: "router-default.apps.test.local",
			},
		},
	}
}

// NewCentralDeployment creates a new k8s Deployment in a given namespace
func NewCentralDeployment(ns string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central",
			Namespace: ns,
			Labels:    centralLabels,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
}
