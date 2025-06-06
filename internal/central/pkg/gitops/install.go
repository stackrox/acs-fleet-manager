package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/golang/glog"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sretry "k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace                = "openshift-gitops-operator"
	argoCdNamespace                  = "openshift-gitops"
	operatorSubscriptionName         = "openshift-gitops-operator"
	operatorGroupName                = "openshift-gitops-operator"
	managedByArgoCdLabelKey          = "argocd.argoproj.io/managed-by"
	managedByArgoCdLabelValue        = operatorNamespace
	awsSecretsManagerMaxBackoffDelay = 5 * time.Second // pragma: allowlist secret
	awsSecretsManagerMaxAttempts     = 3
	awsRepositorySecretID            = "gitops" // pragma: allowlist secret
	bootstrapAppName                 = "rhacs-bootstrap"
	bootstrapAppPath                 = "bootstrap"
	bootstrapAppRepositoryName       = "acscs-manifests-repo"
	bootstrapAppRepositoryURL        = "https://github.com/stackrox/acscs-manifests"
	crdPollInterval                  = 5 * time.Second
)

// InstallGitopsOperator installs an instance of openshift-gitops operator
func InstallGitopsOperator(ctx context.Context, optionsFunc ...InstallOptionsFunc) error {
	opts := &InstallOptions{}
	for _, fn := range optionsFunc {
		fn(opts)
	}

	installer := &operatorInstaller{
		k8sClient:               createK8sClientOrDie(),
		awsSecretsManagerClient: createAwsSecretsManagerClientOrDie(ctx),
		opts:                    *opts,
	}
	return installer.install(ctx)
}

// InstallOptions options for installing the gitops operator
type InstallOptions struct {
	ClusterName                string
	BootstrapAppTargetRevision string
}

// InstallOptionsFunc install option function
type InstallOptionsFunc func(*InstallOptions)

// WithClusterName sets for cluster name in InstallOptions
func WithClusterName(clusterName string) InstallOptionsFunc {
	return func(o *InstallOptions) {
		o.ClusterName = clusterName
	}
}

// WithBootstrapAppTargetRevision sets bootstrap app target revision in InstallOptions
func WithBootstrapAppTargetRevision(revision string) InstallOptionsFunc {
	return func(o *InstallOptions) {
		o.BootstrapAppTargetRevision = revision
	}
}

type operatorInstaller struct {
	k8sClient               ctrlClient.Client
	awsSecretsManagerClient secretsManagerClient
	opts                    InstallOptions
}

func createK8sClientOrDie() ctrlClient.Client {
	config, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatal("failed to get k8s client config", err)
	}
	scheme := runtime.NewScheme()
	utilRuntime.Must(clientgoscheme.AddToScheme(scheme))
	utilRuntime.Must(argoCd.AddToScheme(scheme))
	utilRuntime.Must(operatorsv1alpha1.AddToScheme(scheme))
	utilRuntime.Must(operatorsv1.AddToScheme(scheme))
	utilRuntime.Must(configv1.AddToScheme(scheme))

	k8sClient, err := ctrlClient.New(config, ctrlClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		glog.Fatal("failed to create k8s client", err)
	}

	glog.Infof("Connected to k8s cluster: %s", config.Host)
	return k8sClient
}

//go:generate moq -rm -out secrets_manager_client_moq.go . secretsManagerClient
type secretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func createAwsSecretsManagerClientOrDie(ctx context.Context) secretsManagerClient {
	retryerWithBackoff := retry.AddWithMaxBackoffDelay(retry.NewStandard(), awsSecretsManagerMaxBackoffDelay)
	awsRetryer := func() aws.Retryer {
		return retry.AddWithMaxAttempts(retryerWithBackoff, awsSecretsManagerMaxAttempts)
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRetryer(awsRetryer),
	)
	if err != nil {
		glog.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	return secretsmanager.NewFromConfig(cfg)
}

func (i *operatorInstaller) install(ctx context.Context) error {
	if err := i.ensureNamespace(ctx, operatorNamespace); err != nil {
		return err
	}
	if err := i.ensureNamespace(ctx, argoCdNamespace); err != nil {
		return err
	}
	if err := i.ensureOperatorGroup(ctx); err != nil {
		return err
	}
	if err := i.ensureSubscription(ctx); err != nil {
		return err
	}
	if err := i.ensureRepositorySecret(ctx); err != nil {
		return err
	}
	if err := i.waitForApplicationCRD(ctx); err != nil {
		return err
	}
	return i.ensureBootstrapApplication(ctx)
}

func (i *operatorInstaller) ensureNamespace(ctx context.Context, name string) error {
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
	glog.Infof("Setting '%s=%s' label for namespace %q", managedByArgoCdLabelKey, managedByArgoCdLabelValue, name)
	updateErr := k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		currentNs := &corev1.Namespace{}
		if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: name}, currentNs); err != nil {
			glog.Errorf("Failed to re-fetch namespace %q for update: %v", name, err)
			return fmt.Errorf("failed to re-fetch namespace %q for update: %w", name, err)
		}
		if currentNs.Labels == nil {
			currentNs.Labels = make(map[string]string)
		}
		currentNs.Labels[managedByArgoCdLabelKey] = managedByArgoCdLabelValue
		glog.V(2).Infof("Attempting to update namespace %q (ResourceVersion: %s)", name, currentNs.ResourceVersion)
		return i.k8sClient.Update(ctx, currentNs)
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

func (i *operatorInstaller) getNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: name}, &namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return &namespace, nil
}

func (i *operatorInstaller) ensureSubscription(ctx context.Context) error {
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

func (i *operatorInstaller) ensureOperatorGroup(ctx context.Context) error {
	var operatorGroup operatorsv1.OperatorGroup
	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: operatorGroupName, Namespace: operatorNamespace}, &operatorGroup); err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Info("Openshift Gitops OperatorGroup not found. Creating...")
			if err := i.k8sClient.Create(ctx, newOperatorGroup()); err != nil {
				return fmt.Errorf("creating openshift gitops operator group: %w", err)
			}
			glog.Info("Openshift Gitops OperatorGroup created.")
			return nil
		}
		return fmt.Errorf("getting openshift gitops operator group: %w", err)
	}
	glog.Info("Openshift Gitops OperatorGroup already exists. No update needed.")
	return nil
}

func newOperatorGroup() *operatorsv1.OperatorGroup {
	return &operatorsv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: operatorNamespace,
		},
	}
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

func (i *operatorInstaller) ensureRepositorySecret(ctx context.Context) error {
	glog.Infof("Ensuring repository secret '%s/%s' by fetching from AWS secret %q", argoCdNamespace, bootstrapAppRepositoryName, awsRepositorySecretID)
	foundK8sSecret := &corev1.Secret{}

	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, foundK8sSecret); err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Infof("Kubernetes secret '%s/%s' not found. Creating...", argoCdNamespace, bootstrapAppRepositoryName)
			awsSecretValue, err := i.fetchSecretValueFromAWS(ctx)
			if err != nil {
				return err
			}
			if errCreate := i.k8sClient.Create(ctx, i.newRepositorySecret(awsSecretValue)); errCreate != nil {
				glog.Errorf("Failed to create Kubernetes secret '%s/%s': %v", argoCdNamespace, bootstrapAppRepositoryName, errCreate)
				return fmt.Errorf("creating kubernetes secret '%s/%s': %w", argoCdNamespace, bootstrapAppRepositoryName, errCreate)
			}
			glog.Infof("Successfully created Kubernetes secret '%s/%s'.", argoCdNamespace, bootstrapAppRepositoryName)
			return nil
		}
		// Other error fetching the secret
		glog.Errorf("Failed to get Kubernetes secret '%s/%s': %v", argoCdNamespace, bootstrapAppRepositoryName, err)
		return fmt.Errorf("getting kubernetes secret '%s/%s': %w", argoCdNamespace, bootstrapAppRepositoryName, err)
	}
	awsSecretValue, err := i.fetchSecretValueFromAWS(ctx)
	if err != nil {
		return err
	}

	if i.secretNeedsUpdate(foundK8sSecret, awsSecretValue) {
		glog.Infof("Kubernetes secret '%s/%s' exists but needs update.", argoCdNamespace, bootstrapAppRepositoryName)
		newRepositorySecret := i.newRepositorySecret(awsSecretValue)
		updateErr := k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
			currentSecret := &corev1.Secret{}
			if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, currentSecret); err != nil {
				glog.Errorf("Failed to re-fetch secret '%s/%s' for update: %v", argoCdNamespace, bootstrapAppRepositoryName, err)
				return fmt.Errorf("failed to re-fetch secret '%s/%s' for update: %w", argoCdNamespace, bootstrapAppRepositoryName, err)
			}
			currentSecret.Labels = newRepositorySecret.Labels
			currentSecret.StringData = newRepositorySecret.StringData
			currentSecret.Type = newRepositorySecret.Type
			glog.V(2).Infof("Attempting to update secret '%s/%s' (ResourceVersion: %s)",
				argoCdNamespace, bootstrapAppRepositoryName, currentSecret.ResourceVersion)
			return i.k8sClient.Update(ctx, currentSecret)
		})

		if updateErr != nil {
			glog.Errorf("Failed to update Kubernetes secret '%s/%s': %v", argoCdNamespace, bootstrapAppRepositoryName, updateErr)
			return fmt.Errorf("updating kubernetes secret '%s/%s': %w", argoCdNamespace, bootstrapAppRepositoryName, updateErr)
		}
		glog.Infof("Successfully updated Kubernetes secret '%s/%s'.", argoCdNamespace, bootstrapAppRepositoryName)
	} else {
		glog.Infof("Kubernetes secret '%s/%s' already exists and is up-to-date.", argoCdNamespace, bootstrapAppRepositoryName)
	}

	return nil
}

func (i *operatorInstaller) secretNeedsUpdate(foundSecret *corev1.Secret, awsRepositorySecret awsRepositorySecret) bool {
	if len(foundSecret.OwnerReferences) != 0 {
		// foundSecret is owned by another controller (e.g. ESO), skipping update
		return false
	}
	if foundSecret.Data == nil {
		return true
	}
	urlBytes, ok := foundSecret.Data["url"]
	if !ok || string(urlBytes) != bootstrapAppRepositoryURL {
		return true
	}
	passwordBytes, ok := foundSecret.Data["password"]
	return !ok || string(passwordBytes) != awsRepositorySecret.GithubToken
}

func (i *operatorInstaller) newRepositorySecret(awsRepositorySecret awsRepositorySecret) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: argoCdNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":   "acsfleetctl",
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"url":      bootstrapAppRepositoryURL,
			"username": "not-used",
			"password": awsRepositorySecret.GithubToken,
		},
	}
}

type awsRepositorySecret struct {
	GithubToken string `json:"github_token"`
}

func (i *operatorInstaller) fetchSecretValueFromAWS(ctx context.Context) (awsRepositorySecret, error) {
	glog.Infof("Fetching secret %q from AWS Secrets Manager", awsRepositorySecretID)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(awsRepositorySecretID),
	}
	var secret awsRepositorySecret
	result, err := i.awsSecretsManagerClient.GetSecretValue(ctx, input)
	if err != nil {
		glog.Errorf("Failed to retrieve secret value for %q from AWS: %v", awsRepositorySecretID, err)
		return secret, fmt.Errorf("aws GetSecretValue for %q: %w", awsRepositorySecretID, err)
	}

	if result.SecretString == nil { // pragma: allowlist secret
		return secret, fmt.Errorf("aws secret %q does not contain a SecretString", awsRepositorySecretID)
	}
	err = json.Unmarshal([]byte(*result.SecretString), &secret)
	if err != nil {
		glog.Errorf("Failed to unmarshal JSON secret string from AWS secret %q: %v", awsRepositorySecretID, err)
		return secret, fmt.Errorf("unmarshalling JSON from AWS secret %q: %w", awsRepositorySecretID, err)
	}
	return secret, nil
}

func (i *operatorInstaller) waitForApplicationCRD(ctx context.Context) error {
	glog.Info("Waiting for ArgoCD Application CRD to become available...")
	err := wait.PollUntilContextCancel(ctx, crdPollInterval, true, func(ctx context.Context) (bool, error) {
		appList := &argoCd.ApplicationList{}
		err := i.k8sClient.List(ctx, appList, ctrlClient.InNamespace(argoCdNamespace), ctrlClient.Limit(1))
		if err != nil {
			if meta.IsNoMatchError(err) || runtime.IsNotRegisteredError(err) || apiErrors.IsNotFound(err) {
				glog.V(2).Infof("Application CRD not yet available, retrying: %v", err)
				return false, nil
			}
			glog.Errorf("Error listing Applications while waiting for CRD: %v", err)
			return false, fmt.Errorf("listing Applications while waiting for CRD: %w", err)
		}
		glog.Info("ArgoCD Application CRD is available.")
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("waiting for ArgoCD Application CRD to become available: %w", err)
	}
	return nil
}

func newBootstrapApplication(clusterName string, bootstrapAppTargetRevision string) *argoCd.Application {
	return &argoCd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppName,
			Namespace: argoCdNamespace,
		},
		Spec: argoCd.ApplicationSpec{
			Project: "default",
			Source: &argoCd.ApplicationSource{
				RepoURL:        bootstrapAppRepositoryURL,
				Path:           bootstrapAppPath + "/" + clusterName,
				TargetRevision: bootstrapAppTargetRevision,
			},
			Destination: argoCd.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: argoCdNamespace,
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

func (i *operatorInstaller) ensureBootstrapApplication(ctx context.Context) error {
	glog.Infof("Ensuring bootstrap ArgoCD application %q in namespace %q", bootstrapAppName, argoCdNamespace)

	app := &argoCd.Application{}
	err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: bootstrapAppName, Namespace: argoCdNamespace}, app)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Infof("Bootstrap application %q not found. Creating...", bootstrapAppName)
			clusterName, err := i.getClusterName(ctx)
			if err != nil {
				return fmt.Errorf("error resolving cluster name: %w", err)
			}
			bootstrapApp := newBootstrapApplication(clusterName, i.getBootstrapAppTargetRevision())
			if errCreate := i.k8sClient.Create(ctx, bootstrapApp); errCreate != nil {
				glog.Errorf("Failed to create bootstrap application %q: %v", bootstrapAppName, errCreate)
				return fmt.Errorf("creating bootstrap application %q: %w", bootstrapAppName, errCreate)
			}
			glog.Infof("Bootstrap application %q created successfully.", bootstrapAppName)
			return nil
		}
		glog.Errorf("Failed to get bootstrap application %q: %v", bootstrapAppName, err)
		return fmt.Errorf("getting bootstrap application %q: %w", bootstrapAppName, err)
	}

	glog.Infof("Bootstrap application %q already exists. Skipping creation.", bootstrapAppName)
	return nil
}

func (i *operatorInstaller) resolveClusterName(ctx context.Context) (string, error) {
	infra := &configv1.Infrastructure{}
	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: "cluster"}, infra); err != nil {
		return "", fmt.Errorf("getting infrastructure: %w", err)
	}
	if infra.Status.InfrastructureName == "" {
		return "", fmt.Errorf("infrastructure name is empty the in status of CR")
	}
	return trimSuffixIfExists(infra.Status.InfrastructureName), nil
}

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

func (i *operatorInstaller) getClusterName(ctx context.Context) (string, error) {
	if i.opts.ClusterName == "" {
		return i.resolveClusterName(ctx)
	}
	return i.opts.ClusterName, nil
}

func (i *operatorInstaller) getBootstrapAppTargetRevision() string {
	if i.opts.BootstrapAppTargetRevision == "" {
		return "HEAD"
	}
	return i.opts.BootstrapAppTargetRevision
}
