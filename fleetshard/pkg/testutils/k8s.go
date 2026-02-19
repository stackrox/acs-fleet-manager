// Package testutils ...
package testutils

import (
	"encoding/json"
	"fmt"
	"testing"

	argoCd "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/hashicorp/go-multierror"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sTesting "k8s.io/client-go/testing"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	argoAppGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}
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
	CentralCA                   = "test CA"
	centralReencryptRouteName   = "managed-central-reencrypt"
	centralPassthroughRouteName = "managed-central-passthrough"
)

var centralLabels = map[string]string{
	"app.kubernetes.io/name":      "stackrox",
	"app.kubernetes.io/component": "central",
}

var (
	_ k8sTesting.ObjectTracker = (*ReconcileTracker)(nil)
)

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
func NewFakeClientWithTracker(t *testing.T, objects ...ctrlClient.Object) (ctrlClient.WithWatch, *ReconcileTracker) {
	scheme := NewScheme(t)
	tracker := NewReconcileTracker(scheme)
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjectTracker(tracker).
		WithObjects(objects...).
		Build()
	return client, tracker
}

// NewScheme returns a new scheme instance used for fleetshard testing
func NewScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(scheme))
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, openshiftRouteV1.Install(scheme))
	require.NoError(t, openshiftOperatorV1.Install(scheme))
	require.NoError(t, argoCd.AddToScheme(scheme))

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
func (t *ReconcileTracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string, _ ...metav1.CreateOptions) error {
	if err := t.ObjectTracker.Create(gvr, obj, ns); err != nil {
		return fmt.Errorf("adding GVR %q to reconcile tracker: %w", gvr, err)
	}
	if gvr == argoAppGVR {
		tenantResources, err := NewTenantResources(obj.(*argoCd.Application))
		if err != nil {
			return fmt.Errorf("create tenant resources from ArgoCD app: %w", err)
		}
		var multiErr *multierror.Error
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(centralsGVR, tenantResources.CentralCR, tenantResources.CentralCR.Namespace))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(secretsGVR, tenantResources.CentralTLSSecret, tenantResources.CentralTLSSecret.Namespace))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Create(deploymentGVR, tenantResources.CentralDeployment, tenantResources.CentralDeployment.Namespace))
		multiErr = multierror.Append(multiErr, t.createRoute(tenantResources.CentralReencryptRoute))
		multiErr = multierror.Append(multiErr, t.createRoute(tenantResources.CentralPassthroughRoute))

		if err := multiErr.ErrorOrNil(); err != nil {
			return fmt.Errorf("creating group version resource: %w", err)
		}
	}
	return nil

}

// TenantResources is a container for objects expected to be created by ArgoCD
type TenantResources struct {
	CentralCR               *platform.Central
	CentralTLSSecret        *coreV1.Secret
	CentralDeployment       *appsv1.Deployment
	CentralReencryptRoute   *openshiftRouteV1.Route
	CentralPassthroughRoute *openshiftRouteV1.Route
}

// NewTenantResources creates a new instance of TenantResources
func NewTenantResources(appCR *argoCd.Application) (*TenantResources, error) {
	app, err := newArgoCDApplicationFromCustomResource(appCR)
	if err != nil {
		return nil, fmt.Errorf("parsing ArgoCD application from Custom Resource: %w", err)
	}

	return &TenantResources{
		CentralCR:               centralCrFromArgoCdApp(app),
		CentralTLSSecret:        newCentralTLSSecret(app.destinationNamespace),
		CentralDeployment:       newCentralDeployment(app.destinationNamespace),
		CentralReencryptRoute:   newReencryptRoute(app),
		CentralPassthroughRoute: newPassthroughRoute(app),
	}, nil
}

// Objects returns the list of kubernetes objects that make up the tenant-resources ArgoCD application
func (t *TenantResources) Objects() []ctrlClient.Object {
	return []ctrlClient.Object{
		t.CentralCR,
		t.CentralTLSSecret,
		t.CentralDeployment,
		t.CentralReencryptRoute,
		t.CentralPassthroughRoute,
	}
}

// Update updates an existing object in the tracker in the specified namespace.
func (t *ReconcileTracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string, _ ...metav1.UpdateOptions) error {
	if gvr == argoAppGVR {
		tenantResources, err := NewTenantResources(obj.(*argoCd.Application))
		if err != nil {
			return fmt.Errorf("create tenant resources from ArgoCD app: %w", err)
		}
		var multiErr *multierror.Error
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Update(centralsGVR, tenantResources.CentralCR, tenantResources.CentralCR.Namespace))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Update(secretsGVR, tenantResources.CentralTLSSecret, tenantResources.CentralTLSSecret.Namespace))
		multiErr = multierror.Append(multiErr, t.ObjectTracker.Update(deploymentGVR, tenantResources.CentralDeployment, tenantResources.CentralDeployment.Namespace))
		multiErr = multierror.Append(multiErr, t.updateRoute(tenantResources.CentralReencryptRoute))
		multiErr = multierror.Append(multiErr, t.updateRoute(tenantResources.CentralPassthroughRoute))

		if err := multiErr.ErrorOrNil(); err != nil {
			return fmt.Errorf("creating group version resource: %w", err)
		}
	}
	if err := t.ObjectTracker.Update(gvr, obj, ns); err != nil {
		return fmt.Errorf("adding GVR %q to reconcile tracker: %w", gvr, err)
	}
	return nil
}

func centralCrFromArgoCdApp(app *argoCDApplication) *platform.Central {
	return &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.instanceName(),
			Namespace: app.destinationNamespace,
		},
	}
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
	route.Status = t.admittedStatus(name, route.Spec.Host)
	err := t.ObjectTracker.Create(routesGVR, route, route.GetNamespace())
	return errors.Wrapf(err, "create route")
}

func (t *ReconcileTracker) updateRoute(route *openshiftRouteV1.Route) error {
	name := route.GetName()
	if err := t.routeErrors[name]; err != nil {
		return err
	}
	if t.skipRoute[name] {
		return nil
	}
	route.Status = t.admittedStatus(name, route.Spec.Host)
	err := t.ObjectTracker.Update(routesGVR, route, route.GetNamespace())
	return errors.Wrapf(err, "update route")
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

// newCentralDeployment creates a new k8s Deployment in a given namespace
func newCentralDeployment(ns string) *appsv1.Deployment {
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

func newRoute(name string, namespace string, host string) *openshiftRouteV1.Route {
	return &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
}

func newReencryptRoute(app *argoCDApplication) *openshiftRouteV1.Route {
	route := newRoute(centralReencryptRouteName, app.destinationNamespace, app.centralUIHost())
	route.Spec.TLS = &openshiftRouteV1.TLSConfig{
		Termination:              openshiftRouteV1.TLSTerminationReencrypt,
		DestinationCACertificate: CentralCA,
	}
	return route
}

func newPassthroughRoute(app *argoCDApplication) *openshiftRouteV1.Route {
	route := newRoute(centralPassthroughRouteName, app.destinationNamespace, app.centralDataHost())
	route.Spec.TLS = &openshiftRouteV1.TLSConfig{
		Termination: openshiftRouteV1.TLSTerminationPassthrough,
	}
	return route
}

type argoCDApplication struct {
	destinationNamespace string
	helmValues           map[string]interface{}
}

func (a *argoCDApplication) centralUIHost() string {
	host, ok := a.helmValues["centralUIHost"]
	if !ok {
		return ""
	}
	return host.(string)
}

func (a *argoCDApplication) centralDataHost() string {
	host, ok := a.helmValues["centralDataHost"]
	if !ok {
		return ""
	}
	return host.(string)
}

func (a *argoCDApplication) instanceName() string {
	return a.helmValues["instanceName"].(string)
}

func newArgoCDApplicationFromCustomResource(app *argoCd.Application) (*argoCDApplication, error) {
	helmValues := map[string]interface{}{}
	if err := json.Unmarshal(app.Spec.Source.Helm.ValuesObject.Raw, &helmValues); err != nil {
		return nil, fmt.Errorf("unmarshalling helm values: %w", err)
	}
	return &argoCDApplication{
		destinationNamespace: app.Spec.Destination.Namespace,
		helmValues:           helmValues,
	}, nil
}
