// Package k8s ...
package k8s

import (
	"github.com/golang/glog"
	"github.com/openshift/addon-operator/apis/addons"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	verticalpodautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/dynamic"
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
	_ = addons.AddToScheme(scheme)
	_ = verticalpodautoscalingv1.AddToScheme(scheme)

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

func CreateDynamicClientOrDie() dynamic.Interface {
	config, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatal("failed to get k8s client config", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		glog.Fatal("failed to create dynamic client", err)
	}

	glog.Infof("Connected to k8s cluster: %s", config.Host)
	return dynamicClient
}

// IsRoutesResourceEnabled checks if routes resource are available on the cluster.
func IsRoutesResourceEnabled(client ctrlClient.Client) (bool, error) {
	_, err := client.RESTMapper().ResourceFor(routesGVK)
	if err != nil {
		return false, nil
	}
	return true, nil
}
