// Package k8s ...
package k8s

import (
	"github.com/golang/glog"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	argoCd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	verticalpodautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var routesGVK = schema.GroupVersionResource{
	Group:    "route.openshift.io",
	Version:  "v1",
	Resource: "routes",
}

var CentralGVK = schema.GroupVersionKind{
	Kind:    "Central",
	Group:   "platform.stackrox.io",
	Version: "v1alpha1",
}

func must(err error) {
	if err != nil {
		glog.Fatal(err)
	}
}

// CreateClientOrDie creates a new kubernetes client with default config loader or dies
func CreateClientOrDie() ctrlClient.Client {
	config, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatal("failed to get k8s client config", err)
	}
	return buildClientOrDie(config)
}

// CreateClientWithConfigOrDie create a new kubernetes client with given config
func CreateClientWithConfigOrDie(config *rest.Config) ctrlClient.Client {
	return buildClientOrDie(config)
}

func buildClientOrDie(config *rest.Config) ctrlClient.Client {
	scheme := runtime.NewScheme()
	must(clientgoscheme.AddToScheme(scheme))
	must(v1alpha1.AddToScheme(scheme))
	must(openshiftRouteV1.Install(scheme))
	must(openshiftOperatorV1.Install(scheme))
	must(verticalpodautoscalingv1.AddToScheme(scheme))
	must(argoCd.AddToScheme(scheme))

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

// CreateInterfaceOrDie create new kubernetes interface or dies
func CreateInterfaceOrDie() kubernetes.Interface {

	config, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatal("failed to get k8s client config", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal("error creating clientset: %s", err.Error())
	}

	return clientset
}
