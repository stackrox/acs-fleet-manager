package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config" // Renamed to avoid conflict with ctrl.GetConfig
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
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
	operatorNamespace                = "openshift-gitops-operator"
	argoCdNamespace                  = "openshift-gitops"
	operatorSubscriptionName         = "openshift-gitops-operator"
	managedByArgoCdLabelKey          = "argocd.argoproj.io/managed-by"
	managedByArgoCdLabelValue        = operatorNamespace
	argoCdRepositoryName             = "acscs-manifests-repo"
	argoCdRepositoryURL              = "https://github.com/stackrox/acscs-manifests"
	awsSecretsManagerMaxBackoffDelay = 5 * time.Second // pragma: allowlist secret
	awsSecretsManagerMaxAttempts     = 3
	awsRepositorySecretID            = "gitops" // pragma: allowlist secret
	bootstrapAppName                 = "rhacs-bootstrap"
)

// InstallGitopsOperator installs an instance of openshift-gitops operator
func InstallGitopsOperator(ctx context.Context) error {
	installer := &operatorInstaller{
		k8sClient:               createK8sClientOrDie(),
		awsSecretsManagerClient: createAwsSecretsManagerClientOrDie(ctx),
	}
	return installer.install(ctx)
}

type operatorInstaller struct {
	k8sClient               ctrlClient.Client
	awsSecretsManagerClient secretsManagerClient
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
	if err := i.ensureSubscription(ctx); err != nil {
		return err
	}
	return i.ensureRepositorySecret(ctx)
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
	glog.Infof("Ensuring repository secret '%s/%s' by fetching from AWS secret %q", argoCdNamespace, argoCdRepositoryName, awsRepositorySecretID)
	foundK8sSecret := &corev1.Secret{}

	if err := i.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: argoCdRepositoryName, Namespace: argoCdNamespace}, foundK8sSecret); err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Infof("Kubernetes secret '%s/%s' not found. Creating...", argoCdNamespace, argoCdRepositoryName)
			awsSecretValue, err := i.fetchSecretValueFromAWS(ctx)
			if err != nil {
				return err
			}
			if errCreate := i.k8sClient.Create(ctx, i.newRepositorySecret(awsSecretValue)); errCreate != nil {
				glog.Errorf("Failed to create Kubernetes secret '%s/%s': %v", argoCdNamespace, argoCdRepositoryName, errCreate)
				return fmt.Errorf("creating kubernetes secret '%s/%s': %w", argoCdNamespace, argoCdRepositoryName, errCreate)
			}
			glog.Infof("Successfully created Kubernetes secret '%s/%s'.", argoCdNamespace, argoCdRepositoryName)
			return nil
		}
		// Other error fetching the secret
		glog.Errorf("Failed to get Kubernetes secret '%s/%s': %v", argoCdNamespace, argoCdRepositoryName, err)
		return fmt.Errorf("getting kubernetes secret '%s/%s': %w", argoCdNamespace, argoCdRepositoryName, err)
	}
	awsSecretValue, err := i.fetchSecretValueFromAWS(ctx)
	if err != nil {
		return err
	}

	if i.secretNeedsUpdate(foundK8sSecret, awsSecretValue) {
		glog.Infof("Kubernetes secret '%s/%s' exists but needs update.", argoCdNamespace, argoCdRepositoryName)
		newRepositorySecret := i.newRepositorySecret(awsSecretValue)
		foundK8sSecret.StringData = newRepositorySecret.StringData
		foundK8sSecret.Type = newRepositorySecret.Type

		if errUpdate := i.k8sClient.Update(ctx, foundK8sSecret); errUpdate != nil {
			glog.Errorf("Failed to update Kubernetes secret '%s/%s': %v", argoCdNamespace, argoCdRepositoryName, errUpdate)
			return fmt.Errorf("updating kubernetes secret '%s/%s': %w", argoCdNamespace, argoCdRepositoryName, errUpdate)
		}
		glog.Infof("Successfully updated Kubernetes secret '%s/%s'.", argoCdNamespace, argoCdRepositoryName)
	} else {
		glog.Infof("Kubernetes secret '%s/%s' already exists and is up-to-date.", argoCdNamespace, argoCdRepositoryName)
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
	if !ok || string(urlBytes) != argoCdRepositoryURL {
		return true
	}
	passwordBytes, ok := foundSecret.Data["password"]
	return !ok || string(passwordBytes) != awsRepositorySecret.GithubToken
}

func (i *operatorInstaller) newRepositorySecret(awsRepositorySecret awsRepositorySecret) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      argoCdRepositoryName,
			Namespace: argoCdNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":   "acsfleetctl",
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"url":      argoCdRepositoryURL,
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
