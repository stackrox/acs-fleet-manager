package centralreconciler

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

var routesGVK = schema.GroupVersionResource{
	Group:    "route.openshift.io",
	Version:  "v1",
	Resource: "routes"}

// NewKubeClient creates a new instance of client-go kubernetes.Interface
func NewKubeClient() (client kubernetes.Interface, err error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return client, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, err
	}

	return clientset, err
}

func IsRoutesResourceEnabled() (bool, error) {
	kubeClient, err := NewKubeClient()
	if err != nil {
		return false, errors.Wrapf(err, "create client-go k8s client")
	}
	return discovery.IsResourceEnabled(kubeClient.Discovery(), routesGVK)
}
