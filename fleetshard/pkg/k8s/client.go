package k8s

import (
	"github.com/golang/glog"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateClientOrDie creates a new kubernetes client or dies
func CreateClientOrDie() ctrlClient.WithWatch {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	config := ctrl.GetConfigOrDie()

	k8sClient, err := ctrlClient.NewWithWatch(config, ctrlClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		glog.Fatal("failed to create k8s client", err)
	}

	glog.Infof("Connected to k8s cluster: %s", config.Host)
	return k8sClient
}
