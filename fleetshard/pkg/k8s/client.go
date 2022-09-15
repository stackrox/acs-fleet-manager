package k8s

import (
	"github.com/golang/glog"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var routesGVK = schema.GroupVersionResource{
	Group:    "route.openshift.io",
	Version:  "v1",
	Resource: "routes",
}

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

// IsRoutesResourceEnabled checks if routes resource are available on the cluster.
func IsRoutesResourceEnabled(client ctrlClient.Client) (bool, error) {
	_, err := client.RESTMapper().ResourceFor(routesGVK)
	if err != nil {
		return false, nil
	}
	return true, nil
}
