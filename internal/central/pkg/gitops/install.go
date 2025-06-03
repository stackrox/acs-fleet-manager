package gitops

import (
	"context"
	"fmt"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/golang/glog"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace         = "openshift-gitops-operator"
	argoCdNamespace           = "openshift-gitops"
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
	utilRuntime.Must(operatorsv1alpha1.AddToScheme(scheme))

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
	if err := i.ensureNamespace(ctx, argoCdNamespace); err != nil {
		return err
	}
	return i.ensureSubscription(ctx)
}

func (i *selfManagedOperatorInstaller) ensureNamespace(ctx context.Context, name string) error {
	namespace, err := i.getNamespace(ctx, name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Infof("Namespace %q not found. Creating...", name)
			namespace = newNamespace(name)
			if err := i.k8sClient.Create(ctx, namespace); err != nil {
				return fmt.Errorf("creating namespace %q: %w", name, err)
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

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func (i *selfManagedOperatorInstaller) getNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: name}, &namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return &namespace, nil
}

func (i *selfManagedOperatorInstaller) ensureSubscription(ctx context.Context) error {
	var subscription operatorsv1alpha1.Subscription
	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: operatorSubscriptionName, Namespace: operatorNamespace}, &subscription); err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Info("Openshift Gitops Subscription not found. Creating...")
			if err := i.k8sClient.Create(ctx, newSubscription()); err != nil {
				return fmt.Errorf("creating openshift gitops subscription: %w", err)
			}
			glog.Info("Openshift Gitops Subscription created.")
			return nil
		}
		return fmt.Errorf("getting openshift gitops subscription: %w", err)
	}
	glog.Info("Openshift Gitops Subscription already exists. No update needed.")
	return nil
}

func newSubscription() *operatorsv1alpha1.Subscription {
	return &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorSubscriptionName,
			Namespace: operatorNamespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			Channel:                "latest",
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Package:                "openshift-gitops-operator",
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
		},
	}
}
