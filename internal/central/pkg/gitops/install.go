package gitops

import (
	"context"
	"fmt"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace         = "openshift-gitops-operator"
	argoCDNamespace           = "openshift-gitops"
	operatorSubscriptionName  = "openshift-gitops-operator"
	applicationName           = "rhacs-gitops"
	managedByArgoCdLabelKey   = "argocd.argoproj.io/managed-by"
	managedByArgoCdLabelValue = operatorNamespace
)

// InstallSelfManagedGitopsOperator installs a self-managed instance of openshift-gitops operator
func InstallSelfManagedGitopsOperator(ctx context.Context) error {
	return newSelfManagedOperatorInstaller().install(ctx)
}

type selfManagedOperatorInstaller struct {
	k8sClient ctrlClient.Client
}

func newSelfManagedOperatorInstaller() *selfManagedOperatorInstaller {
	return &selfManagedOperatorInstaller{
		k8sClient: createClientOrDie(),
	}
}

func createClientOrDie() ctrlClient.Client {
	config, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatal("failed to get k8s client config", err)
	}
	scheme := runtime.NewScheme()
	utilRuntime.Must(clientgoscheme.AddToScheme(scheme))
	utilRuntime.Must(argoCd.AddToScheme(scheme))

	k8sClient, err := ctrlClient.New(config, ctrlClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		glog.Fatal("failed to create k8s client", err)
	}

	glog.Infof("Connected to k8s cluster: %s", config.Host)
	return k8sClient
}

func (i *selfManagedOperatorInstaller) install(ctx context.Context) error {
	if err := i.ensureNamespace(ctx, operatorNamespace); err != nil {
		return err
	}
	return i.ensureNamespace(ctx, argoCDNamespace)
}

func (i *selfManagedOperatorInstaller) ensureNamespace(ctx context.Context, name string) error {
	namespace, err := i.getNamespace(ctx, name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Infof("Namespace %q not found. Creating...", name)
			if err := i.k8sClient.Create(ctx, namespace); err != nil {
				return fmt.Errorf("creating namespace %q: %w", namespace.Name, err)
			}
			glog.Infof("Namespace %q created.", name)
		} else {
			return fmt.Errorf("getting namespace %q: %w", namespace, err)
		}
	} else {
		glog.Infof("Namespace %q found.", name)
	}
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	if currentValue, ok := namespace.Labels[managedByArgoCdLabelKey]; ok && currentValue == managedByArgoCdLabelValue {
		glog.Infof("Label '%s=%s' already exists on namespace '%s'. No update needed.", managedByArgoCdLabelKey, managedByArgoCdLabelValue, name)
		return nil // No change needed
	}
	namespace.Labels[managedByArgoCdLabelKey] = managedByArgoCdLabelValue
	glog.Infof("Setting '%s=%s' label for namespace %q", managedByArgoCdLabelKey, managedByArgoCdLabelValue, name)
	if err := i.k8sClient.Update(ctx, namespace); err != nil {
		return fmt.Errorf("failed to update namespace '%s': %w", name, err)
	}
	return nil
}

func (i *selfManagedOperatorInstaller) getNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: name}, &namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return &namespace, nil
}
