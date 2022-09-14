package k8s

import (
	"context"
	"github.com/golang/glog"
	openshiftOperatorV1 "github.com/openshift/api/operator/v1"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

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
func IsRoutesResourceEnabled(ctx context.Context, client ctrlClient.Client) (bool, error) {
	err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: "default", Name: "does-not-exist"}, &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{},
	})
	if apiErrors.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, nil
	}
	return true, nil
}
