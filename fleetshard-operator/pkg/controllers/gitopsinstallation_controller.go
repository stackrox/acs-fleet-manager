// Package controllers is responsible for operator controllers
package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/pingcap/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sretry "k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ArgoCdNamespace namespace where an ArgoCD instance is deployed
	ArgoCdNamespace = "openshift-gitops"
	// GitopsOperatorNamespace namespace where the gitops operator is deployed
	GitopsOperatorNamespace = "openshift-gitops-operator"
	gitopsInstallationName  = "rhacs-gitops"

	operatorSubscriptionName  = "openshift-gitops-operator"
	operatorGroupName         = "openshift-gitops-operator"
	managedByArgoCdLabelKey   = "argocd.argoproj.io/managed-by"
	managedByArgoCdLabelValue = GitopsOperatorNamespace

	tokenKey                   = "github-token"
	bootstrapAppName           = "rhacs-bootstrap"
	bootstrapAppPath           = "bootstrap"
	bootstrapAppRepositoryName = "acscs-manifests-repo"
	bootstrapAppRepositoryURL  = "https://github.com/stackrox/acscs-manifests"
	crdPollInterval            = 5 * time.Second
)

var _ reconcile.Reconciler = &GitopsInstallationReconciler{}

// GitopsInstallationReconciler gitops installation reconciler
type GitopsInstallationReconciler struct {
	Client          ctrlClient.Client
	SourceNamespace string
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitopsInstallationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.Add(manager.RunnableFunc(r.createDefaultGitopsInstallation)); err != nil {
		return fmt.Errorf("failed to add the default GitopsInstallation func: %v", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GitopsInstallation{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapGitopsInstallation),
			builder.WithPredicates(r.predicateFuncs(r.matchesSourceRepositorySecret))).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapGitopsInstallation),
			builder.WithPredicates(r.predicateFuncs(r.matchesDestinationRepositorySecret))).
		Watches(&argoCd.Application{},
			handler.EnqueueRequestsFromMapFunc(r.mapGitopsInstallation),
			builder.WithPredicates(r.predicateFuncs(r.matchesBootstrapApp))).
		Complete(r)
}

func (r *GitopsInstallationReconciler) mapGitopsInstallation(_ context.Context, _ ctrlClient.Object) []reconcile.Request {
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      gitopsInstallationName,
				Namespace: r.SourceNamespace,
			},
		},
	}
}

func (r *GitopsInstallationReconciler) predicateFuncs(matchesResource func(namespace, name string) bool) predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return matchesResource(e.ObjectNew.GetNamespace(), e.ObjectNew.GetName()) &&
				e.ObjectNew.GetResourceVersion() != e.ObjectOld.GetResourceVersion()
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return matchesResource(e.Object.GetNamespace(), e.Object.GetName())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return matchesResource(e.Object.GetNamespace(), e.Object.GetName())
		},
	}
}

func (r *GitopsInstallationReconciler) matchesSourceRepositorySecret(namespace, name string) bool {
	return namespace == r.SourceNamespace && bootstrapAppRepositoryName == name
}

func (r *GitopsInstallationReconciler) matchesDestinationRepositorySecret(namespace, name string) bool {
	return namespace == ArgoCdNamespace && bootstrapAppRepositoryName == name
}

func (r *GitopsInstallationReconciler) matchesBootstrapApp(namespace, name string) bool {
	return namespace == ArgoCdNamespace && bootstrapAppName == name

}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GitopsInstallationReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Reconciling GitopsInstallation %s/%s", request.Namespace, request.Name)
	instance := &v1alpha1.GitopsInstallation{}
	err := r.Client.Get(ctx, ctrlClient.ObjectKey{Namespace: request.Namespace, Name: request.Name}, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if err := r.ensureNamespace(ctx, GitopsOperatorNamespace); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureNamespace(ctx, ArgoCdNamespace); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureOperatorGroup(ctx); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureSubscription(ctx); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureRepositorySecret(ctx); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.waitForApplicationCRD(ctx); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.ensureBootstrapApplication(ctx, instance.Spec); err != nil {
		return reconcile.Result{}, err
	}
	glog.Infof("Reconciled GitopsInstallation %s/%s", request.Namespace, request.Name)
	return reconcile.Result{}, nil
}

func (r *GitopsInstallationReconciler) ensureNamespace(ctx context.Context, name string) error {
	namespace, err := r.getNamespace(ctx, name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.V(5).Infof("Namespace %q not found. Creating...", name)
			namespace = newNamespace(name)
			if err := r.Client.Create(ctx, namespace); err != nil {
				return fmt.Errorf("creating namespace %q: %w", name, err)
			}
			glog.V(5).Infof("Namespace %q created.", name)
		} else {
			return fmt.Errorf("getting namespace %q: %w", namespace, err)
		}
	} else {
		glog.V(10).Infof("Namespace %q found.", name)
	}
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	if currentValue, ok := namespace.Labels[managedByArgoCdLabelKey]; ok && currentValue == managedByArgoCdLabelValue {
		glog.V(10).Infof("Label '%s=%s' already exists on namespace '%s'. No update needed.", managedByArgoCdLabelKey, managedByArgoCdLabelValue, name)
		return nil // No change needed
	}
	glog.V(5).Infof("Setting '%s=%s' label for namespace %q", managedByArgoCdLabelKey, managedByArgoCdLabelValue, name)
	updateErr := k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		currentNs := &corev1.Namespace{}
		if err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: name}, currentNs); err != nil {
			glog.Errorf("Failed to re-fetch namespace %q for update: %v", name, err)
			return fmt.Errorf("failed to re-fetch namespace %q for update: %w", name, err)
		}
		if currentNs.Labels == nil {
			currentNs.Labels = make(map[string]string)
		}
		currentNs.Labels[managedByArgoCdLabelKey] = managedByArgoCdLabelValue
		glog.V(10).Infof("Attempting to update namespace %q (ResourceVersion: %s)", name, currentNs.ResourceVersion)
		return r.Client.Update(ctx, currentNs)
	})
	if updateErr != nil {
		glog.Errorf("Failed to update labels for namespace %q after retries: %v", name, updateErr)
		return fmt.Errorf("updating labels for namespace %q: %w", name, updateErr)
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

func (r *GitopsInstallationReconciler) getNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: name}, &namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return &namespace, nil
}

func (r *GitopsInstallationReconciler) ensureSubscription(ctx context.Context) error {
	var subscription operatorsv1alpha1.Subscription
	if err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: operatorSubscriptionName, Namespace: GitopsOperatorNamespace}, &subscription); err != nil {
		if apiErrors.IsNotFound(err) {
			glog.V(5).Info("Openshift Gitops Subscription not found. Creating...")
			if err := r.Client.Create(ctx, newSubscription()); err != nil {
				return fmt.Errorf("creating openshift gitops subscription: %w", err)
			}
			glog.V(5).Info("Openshift Gitops Subscription created.")
			return nil
		}
		return fmt.Errorf("getting openshift gitops subscription: %w", err)
	}
	glog.V(10).Info("Openshift Gitops Subscription already exists. No update needed.")
	return nil
}

func (r *GitopsInstallationReconciler) ensureOperatorGroup(ctx context.Context) error {
	var operatorGroup operatorsv1.OperatorGroup
	if err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: operatorGroupName, Namespace: GitopsOperatorNamespace}, &operatorGroup); err != nil {
		if apiErrors.IsNotFound(err) {
			glog.V(5).Info("Openshift Gitops OperatorGroup not found. Creating...")
			if err := r.Client.Create(ctx, newOperatorGroup()); err != nil {
				return fmt.Errorf("creating openshift gitops operator group: %w", err)
			}
			glog.V(5).Info("Openshift Gitops OperatorGroup created.")
			return nil
		}
		return fmt.Errorf("getting openshift gitops operator group: %w", err)
	}
	glog.V(10).Info("Openshift Gitops OperatorGroup already exists. No update needed.")
	return nil
}

func newOperatorGroup() *operatorsv1.OperatorGroup {
	return &operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: GitopsOperatorNamespace,
		},
	}
}

func newSubscription() *operatorsv1alpha1.Subscription {
	return &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorSubscriptionName,
			Namespace: GitopsOperatorNamespace,
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

func (r *GitopsInstallationReconciler) ensureRepositorySecret(ctx context.Context) error {
	glog.V(10).Infof("Ensuring repository secret '%s/%s' by copying from source secret", ArgoCdNamespace, bootstrapAppRepositoryName)
	sourceSecret := &corev1.Secret{}
	sourceRepositorySecretName := bootstrapAppRepositoryName // pragma: allowlist secret
	err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: sourceRepositorySecretName, Namespace: r.SourceNamespace}, sourceSecret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Errorf("Source secret '%s/%s' not found. Cannot proceed with repository configuration.", r.SourceNamespace, sourceRepositorySecretName)
			return fmt.Errorf("source secret '%s' in namespace '%s' not found: %w", sourceRepositorySecretName, r.SourceNamespace, err)
		}
		glog.Errorf("Failed to get source secret '%s/%s': %v", r.SourceNamespace, sourceRepositorySecretName, err)
		return fmt.Errorf("getting source secret: %w", err)
	}
	tokenBytes, ok := sourceSecret.Data[tokenKey]
	if !ok || len(tokenBytes) == 0 {
		return fmt.Errorf("source secret '%s/%s' does not contain a non-empty key '%s'", r.SourceNamespace, sourceRepositorySecretName, tokenKey)
	}

	desiredSecret := newRepositorySecret(tokenBytes)

	foundSecret := &corev1.Secret{}
	err = r.Client.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace}, foundSecret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.V(5).Infof("Destination repository secret '%s/%s' not found. Creating...", ArgoCdNamespace, bootstrapAppRepositoryName)
			if createErr := r.Client.Create(ctx, desiredSecret); createErr != nil {
				return fmt.Errorf("creating destination secret: %w", createErr)
			}
			glog.V(5).Infof("Destination secret created successfully.")
			return nil
		}
		return fmt.Errorf("getting destination secret: %w", err)
	}

	if !repositorySecretNeedsUpdate(foundSecret, desiredSecret) {
		glog.V(10).Infof("Destination repository secret '%s/%s' is already up-to-date.", ArgoCdNamespace, bootstrapAppRepositoryName)
		return nil
	}

	glog.V(10).Infof("Destination repository secret '%s/%s' needs update. Attempting update...", ArgoCdNamespace, bootstrapAppRepositoryName)
	updateErr := k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		currentSecret := &corev1.Secret{}
		getErr := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace}, currentSecret)
		if getErr != nil {
			return fmt.Errorf("getting destination repository secret: %w", getErr)
		}
		currentSecret.Data = desiredSecret.Data
		currentSecret.Type = desiredSecret.Type
		currentSecret.Labels = desiredSecret.Labels
		return r.Client.Update(ctx, currentSecret)
	})

	if updateErr != nil {
		return fmt.Errorf("updating destination secret after retries: %w", updateErr)
	}

	glog.V(10).Infof("Destination repository secret '%s/%s' updated successfully.", ArgoCdNamespace, bootstrapAppRepositoryName)
	return nil
}

func repositorySecretNeedsUpdate(current, desired *corev1.Secret) bool {
	return !reflect.DeepEqual(current.Data, desired.Data) ||
		!reflect.DeepEqual(current.Labels, desired.Labels) ||
		current.Type != desired.Type
}

func newRepositorySecret(token []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: ArgoCdNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Type: corev1.SecretTypeOpaque,
		// Data must be used here instead of StringData for comparison in needsUpdate
		Data: map[string][]byte{
			"url":      []byte(bootstrapAppRepositoryURL),
			"password": token,
		},
	}
}

func (r *GitopsInstallationReconciler) waitForApplicationCRD(ctx context.Context) error {
	glog.V(10).Info("Waiting for ArgoCD Application CRD to become available...")
	err := wait.PollUntilContextCancel(ctx, crdPollInterval, true, func(ctx context.Context) (bool, error) {
		appList := &argoCd.ApplicationList{}
		err := r.Client.List(ctx, appList, ctrlClient.InNamespace(ArgoCdNamespace), ctrlClient.Limit(1))
		if err != nil {
			if meta.IsNoMatchError(err) || runtime.IsNotRegisteredError(err) || apiErrors.IsNotFound(err) {
				glog.V(10).Infof("Application CRD not yet available, retrying: %v", err)
				return false, nil
			}
			glog.Errorf("Error listing Applications while waiting for CRD: %v", err)
			return false, fmt.Errorf("listing Applications while waiting for CRD: %w", err)
		}
		glog.V(10).Info("ArgoCD Application CRD is available.")
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("waiting for ArgoCD Application CRD to become available: %w", err)
	}
	return nil
}

func newBootstrapApplication(spec v1alpha1.GitopsInstallationSpec) *argoCd.Application {
	return &argoCd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppName,
			Namespace: ArgoCdNamespace,
		},
		Spec: argoCd.ApplicationSpec{
			Project: "default",
			Source: &argoCd.ApplicationSource{
				RepoURL:        bootstrapAppRepositoryURL,
				Path:           bootstrapAppPath + "/" + spec.ClusterName,
				TargetRevision: spec.BootstrapAppTargetRevision,
			},
			Destination: argoCd.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: ArgoCdNamespace,
			},
			SyncPolicy: &argoCd.SyncPolicy{
				Automated: &argoCd.SyncPolicyAutomated{
					Prune:      true,
					SelfHeal:   true,
					AllowEmpty: true,
				},
				Retry: &argoCd.RetryStrategy{
					Limit: -1, // number of failed sync attempt retries; unlimited number of attempts if less than 0
					Backoff: &argoCd.Backoff{
						Duration:    "5s",             // the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")
						Factor:      ptr.To(int64(2)), // a factor to multiply the base duration after each failed retry
						MaxDuration: "3m",             // the maximum amount of time allowed for the backoff strategy
					},
				},
			},
		},
	}
}

func (r *GitopsInstallationReconciler) ensureBootstrapApplication(ctx context.Context, spec v1alpha1.GitopsInstallationSpec) error {
	glog.V(10).Infof("Ensuring bootstrap ArgoCD application %q in namespace %q", bootstrapAppName, ArgoCdNamespace)
	foundApp := &argoCd.Application{}
	desiredApp := newBootstrapApplication(spec)

	err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppName, Namespace: ArgoCdNamespace}, foundApp)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.V(5).Infof("Bootstrap application %q not found. Creating...", bootstrapAppName)
			if errCreate := r.Client.Create(ctx, desiredApp); errCreate != nil {
				glog.Errorf("Failed to create bootstrap application %q: %v", bootstrapAppName, errCreate)
				return fmt.Errorf("creating bootstrap application %q: %w", bootstrapAppName, errCreate)
			}
			glog.V(10).Infof("Bootstrap application %q created successfully.", bootstrapAppName)
			return nil
		}
		glog.Errorf("Failed to get bootstrap application %q: %v", bootstrapAppName, err)
		return fmt.Errorf("getting bootstrap application %q: %w", bootstrapAppName, err)
	}

	if !bootstrapApplicationNeedsUpdate(foundApp, desiredApp) {
		glog.V(10).Infof("Bootstrap Application '%s/%s' is already up-to-date.", ArgoCdNamespace, bootstrapAppName)
		return nil
	}

	glog.V(10).Infof("Bootstrap Application '%s/%s' needs update. Attempting update...", ArgoCdNamespace, bootstrapAppName)
	updateErr := k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		currentApp := &argoCd.Application{}
		getErr := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppName, Namespace: ArgoCdNamespace}, currentApp)
		if getErr != nil {
			return fmt.Errorf("getting bootstrap application: %w", getErr)
		}
		currentApp.Spec = desiredApp.Spec
		currentApp.Labels = desiredApp.Labels
		return r.Client.Update(ctx, currentApp)
	})

	if updateErr != nil {
		return fmt.Errorf("updating destination secret after retries: %w", updateErr)
	}

	glog.V(10).Infof("Bootstrap application '%s/%s' already exists updated successfully.", ArgoCdNamespace, bootstrapAppName)
	return nil
}

func bootstrapApplicationNeedsUpdate(current, desired *argoCd.Application) bool {
	return !reflect.DeepEqual(current.Spec, desired.Spec) ||
		!reflect.DeepEqual(current.Labels, desired.Labels)
}

func (r *GitopsInstallationReconciler) resolveClusterName(ctx context.Context) (string, error) {
	infra := &configv1.Infrastructure{}
	if err := r.Client.Get(ctx, ctrlClient.ObjectKey{Name: "cluster"}, infra); err != nil {
		return "", fmt.Errorf("getting infrastructure: %w", err)
	}
	if infra.Status.InfrastructureName == "" {
		return "", fmt.Errorf("infrastructure name is empty the in status of CR")
	}
	return trimSuffixIfExists(infra.Status.InfrastructureName), nil
}

// trimSuffixIfExists returns the cluster name without an autogenerated suffix.
// If there's no suffix, returns the given string as is.
// Example: for the infrastructure name like "acs-dev-dp-01-ocjtq" it will return acs-dev-dp-01.
func trimSuffixIfExists(str string) string {
	if str == "" {
		return ""
	}
	lastDash := strings.LastIndex(str, "-")
	if lastDash == -1 {
		return str // No dash found, return as is
	}
	// If the dash is the last character, or if it's the only character.
	if lastDash == len(str)-1 || lastDash == 0 {
		// Handle cases like "cluster-" or "-"
		// For "cluster-", we want "cluster". For "-", we want "".
		processed := str[:lastDash]
		return processed
	}

	// Check if the part after the last dash looks like a typical ROSA/OSD suffix (e.g., 3 or 6 random chars)
	// This is a heuristic. A more robust check might involve regex or specific length checks.
	// For simplicity, we'll just take everything before the last dash if a dash exists and is not at the end.
	return str[:lastDash]
}

func (r *GitopsInstallationReconciler) createDefaultGitopsInstallation(ctx context.Context) error {
	clusterName, err := r.resolveClusterName(ctx)
	if err != nil {
		return fmt.Errorf("error resolving cluster name: %w", err)
	}
	instance := &v1alpha1.GitopsInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.SourceNamespace,
			Name:      gitopsInstallationName,
		},
		Spec: v1alpha1.GitopsInstallationSpec{
			ClusterName:                clusterName,
			BootstrapAppTargetRevision: "HEAD",
		},
	}
	if err := r.Client.Create(ctx, instance); err != nil {
		if apiErrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("error creating gitops installation: %w", err)
	}
	return nil
}
