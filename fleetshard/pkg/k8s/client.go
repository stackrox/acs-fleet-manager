package k8s

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var routesGVK = schema.GroupVersionResource{
	Group:    "route.openshift.io",
	Version:  "v1",
	Resource: "routes",
}

// ClientMetricsWrapper is a decorator for a k8s client that decorates each request
// method with incrementing the requests counter for k8s in the metrics package
type ClientMetricsWrapper struct {
	Client ctrlClient.Client
}

// Get wraps the Client Get method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) Get(ctx context.Context, key ctrlClient.ObjectKey, obj ctrlClient.Object) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.Get(ctx, key, obj) //nolint:wrapcheck
}

// List wraps the Client List method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) List(ctx context.Context, list ctrlClient.ObjectList, opts ...ctrlClient.ListOption) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.List(ctx, list, opts...) //nolint:wrapcheck
}

// Create wraps the Client Create method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) Create(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.Create(ctx, obj, opts...) //nolint:wrapcheck
}

// Delete wraps the Client Delete method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) Delete(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.Delete(ctx, obj, opts...) //nolint:wrapcheck
}

// Update wraps the Client Update method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) Update(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.UpdateOption) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.Update(ctx, obj, opts...) //nolint:wrapcheck
}

// Patch wraps the Client Patch method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) Patch(ctx context.Context, obj ctrlClient.Object, patch ctrlClient.Patch, opts ...ctrlClient.PatchOption) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.Patch(ctx, obj, patch, opts...) //nolint:wrapcheck
}

// DeleteAllOf wraps the Client DeleteAllOf method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) DeleteAllOf(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.DeleteAllOfOption) error {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.DeleteAllOf(ctx, obj, opts...) //nolint:wrapcheck
}

// Status wraps the Client Status method with incrementing K8sRequests metric
func (cmw *ClientMetricsWrapper) Status() ctrlClient.StatusWriter {
	fleetshardmetrics.IncrementsK8sRequests()
	return cmw.Client.Status()
}

// Scheme wraps the Client Scheme method
func (cmw *ClientMetricsWrapper) Scheme() *runtime.Scheme {
	return cmw.Client.Scheme()
}

// RESTMapper wraps the Client RESTMapper method
func (cmw *ClientMetricsWrapper) RESTMapper() meta.RESTMapper {
	return cmw.Client.RESTMapper()
}

var _ ctrlClient.Client = &ClientMetricsWrapper{}

// CreateClientOrDie creates a new kubernetes client or dies
func CreateClientOrDie() ctrlClient.Client {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = openshiftRouteV1.Install(scheme)
	_ = openshiftOperatorV1.Install(scheme)

	config, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatal("failed to get k8s client config", err)
	}

	k8sClient, err := ctrlClient.New(config, ctrlClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		glog.Fatal("failed to create k8s client", err)
	}

	glog.Infof("Connected to k8s cluster: %s", config.Host)
	return k8sClient
}

func newClientGoClientSet() (client kubernetes.Interface, err error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return client, fmt.Errorf("retrieving Kubernetes config: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, fmt.Errorf("creating Clientset for Kubernetes config: %w", err)
	}

	return clientSet, nil
}

// IsRoutesResourceEnabled ...
func IsRoutesResourceEnabled() (bool, error) {
	clientSet, err := newClientGoClientSet()
	if err != nil {
		return false, fmt.Errorf("creating Kubernetes Clientset: %w", err)
	}

	enabled, err := discovery.IsResourceEnabled(clientSet.Discovery(), routesGVK)
	if err != nil {
		return enabled, fmt.Errorf("checking availability of resource type %s: %w", routesGVK.String(), err)
	}
	return enabled, nil
}
