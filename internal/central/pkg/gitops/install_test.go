package gitops

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newTestInstallerWithAWSMock creates an installer with a fake K8s client and a mocked AWS SM client
func newTestInstallerWithAWSMock(secretsManagerClient *secretsManagerClientMock, initK8sObjs ...ctrlClient.Object) *operatorInstaller {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = argoCd.AddToScheme(scheme)
	_ = operatorsv1alpha1.AddToScheme(scheme)
	_ = operatorsv1.AddToScheme(scheme)
	_ = configv1.AddToScheme(scheme)

	fakeK8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initK8sObjs...).Build()
	return &operatorInstaller{
		k8sClient:               fakeK8sClient,
		awsSecretsManagerClient: secretsManagerClient,
	}
}

// newTestInstaller is the original helper, retained for tests not needing AWS mock
func newTestInstaller(initK8sObjs ...ctrlClient.Object) *operatorInstaller {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = argoCd.AddToScheme(scheme)
	_ = operatorsv1alpha1.AddToScheme(scheme)
	_ = operatorsv1.AddToScheme(scheme)
	_ = configv1.AddToScheme(scheme)

	fakeK8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initK8sObjs...).Build()
	return &operatorInstaller{
		k8sClient: fakeK8sClient,
	}
}

func TestEnsureNamespace_DoesNotExist(t *testing.T) {
	ctx := context.Background()
	expectedLabelKey := managedByArgoCdLabelKey
	expectedLabelValue := managedByArgoCdLabelValue
	testNsName := "new-ns-test"
	installer := newTestInstaller()

	err := installer.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	createdNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: testNsName}, createdNs)
	require.NoError(t, err, "Namespace should have been created")

	require.NotNil(t, createdNs.Labels, "Labels map should be initialized")
	assert.Equal(t, expectedLabelValue, createdNs.Labels[expectedLabelKey], "Managed-by label should be set correctly")
}

func TestEnsureNamespace_Exists_NoLabels(t *testing.T) {
	ctx := context.Background()
	expectedLabelKey := managedByArgoCdLabelKey
	expectedLabelValue := managedByArgoCdLabelValue
	testNsName := "existing-ns-no-labels"
	initialNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNsName},
	}
	installer := newTestInstaller(initialNs)

	err := installer.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	updatedNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: testNsName}, updatedNs)
	require.NoError(t, err)
	require.NotNil(t, updatedNs.Labels)
	assert.Equal(t, expectedLabelValue, updatedNs.Labels[expectedLabelKey])
}

func TestEnsureNamespace_Exists_CorrectLabelExists(t *testing.T) {
	ctx := context.Background()
	expectedLabelKey := managedByArgoCdLabelKey
	expectedLabelValue := managedByArgoCdLabelValue
	testNsName := "existing-ns-correct-label"
	initialNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNsName,
			Labels: map[string]string{expectedLabelKey: expectedLabelValue},
		},
	}
	installer := newTestInstaller(initialNs)

	err := installer.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	currentNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: testNsName}, currentNs)
	require.NoError(t, err)
	require.NotNil(t, currentNs.Labels)
	assert.Equal(t, expectedLabelValue, currentNs.Labels[expectedLabelKey])
}

func TestEnsureNamespace_Exists_LabelKeyExists_WrongValue(t *testing.T) {
	ctx := context.Background()
	expectedLabelKey := managedByArgoCdLabelKey
	expectedLabelValue := managedByArgoCdLabelValue
	testNsName := "existing-ns-wrong-label-value"
	initialNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNsName,
			Labels: map[string]string{expectedLabelKey: "some-other-value"},
		},
	}
	installer := newTestInstaller(initialNs)

	err := installer.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	updatedNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: testNsName}, updatedNs)
	require.NoError(t, err)
	require.NotNil(t, updatedNs.Labels)
	assert.Equal(t, expectedLabelValue, updatedNs.Labels[expectedLabelKey])
}

func TestEnsureNamespace_Exists_WithOtherLabels(t *testing.T) {
	ctx := context.Background()
	expectedLabelKey := managedByArgoCdLabelKey
	expectedLabelValue := managedByArgoCdLabelValue
	testNsName := "existing-ns-other-labels"
	otherLabelKey := "other.key/foo"
	otherLabelValue := "bar"
	initialNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testNsName,
			Labels: map[string]string{otherLabelKey: otherLabelValue},
		},
	}
	installer := newTestInstaller(initialNs)

	err := installer.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	updatedNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: testNsName}, updatedNs)
	require.NoError(t, err)
	require.NotNil(t, updatedNs.Labels)
	assert.Equal(t, expectedLabelValue, updatedNs.Labels[expectedLabelKey], "Managed-by label should be set")
	assert.Equal(t, otherLabelValue, updatedNs.Labels[otherLabelKey], "Other existing labels should be preserved")
	assert.Len(t, updatedNs.Labels, 2, "Should have two labels")
}

func TestEnsureSubscription_DoesNotExist(t *testing.T) {
	ctx := context.Background()
	installer := newTestInstaller()

	err := installer.ensureSubscription(ctx)
	require.NoError(t, err)

	createdSub := &operatorsv1alpha1.Subscription{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: operatorNamespace}, createdSub)
	require.NoError(t, err, "Subscription should have been created")

	assert.Equal(t, operatorSubscriptionName, createdSub.Name)
	assert.Equal(t, operatorNamespace, createdSub.Namespace)
	require.NotNil(t, createdSub.Spec, "Created subscription spec should not be nil")
}

func TestEnsureSubscription_Exists(t *testing.T) {
	ctx := context.Background()
	existingSub := newSubscription()
	existingSub.Spec.Channel = "stable"

	installer := newTestInstaller(existingSub)

	err := installer.ensureSubscription(ctx)
	require.NoError(t, err)

	currentSub := &operatorsv1alpha1.Subscription{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: operatorNamespace}, currentSub)
	require.NoError(t, err, "Subscription should still exist")

	// The current ensureSubscription logic does not update an existing subscription.
	// It only checks for existence and creates if not found.
	assert.Equal(t, existingSub.Spec.Channel, currentSub.Spec.Channel, "Existing subscription channel should not have changed")
	assert.Equal(t, existingSub.Spec.Package, currentSub.Spec.Package)
}

func TestEnsureRepositorySecret_AWSFetchFails(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient)

	expectedError := errors.New("aws error")
	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return nil, expectedError
	}

	err := installer.ensureRepositorySecret(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aws GetSecretValue")
	assert.Contains(t, err.Error(), expectedError.Error())
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_AWSSecretStringNil(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient)

	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretBinary: []byte("some binary data")}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a SecretString")
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_AWSSecretUnmarshalFails(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient)

	invalidJSONString := "this is not json"
	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(invalidJSONString)}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling JSON from AWS secret")
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_K8sSecretDoesNotExist_CreateSuccess(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient) // No initial K8s objects

	awsSecretData := awsRepositorySecret{GithubToken: "test-token-123"} // pragma: allowlist secret
	awsSecretJSON, _ := json.Marshal(awsSecretData)

	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(awsSecretJSON))}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	createdK8sSecret := &corev1.Secret{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, createdK8sSecret)
	require.NoError(t, err, "Kubernetes secret should have been created")

	assert.Equal(t, bootstrapAppRepositoryURL, createdK8sSecret.StringData["url"])
	assert.Equal(t, "not-used", createdK8sSecret.StringData["username"])
	assert.Equal(t, "test-token-123", createdK8sSecret.StringData["password"])
	assert.Equal(t, "acsfleetctl", createdK8sSecret.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "repository", createdK8sSecret.Labels["argocd.argoproj.io/secret-type"])
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_K8sSecretExists_UpToDate(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}

	awsSecretData := awsRepositorySecret{GithubToken: "current-token"} // pragma: allowlist secret
	awsSecretJSON, _ := json.Marshal(awsSecretData)

	initialK8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: argoCdNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":   "acsfleetctl",
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"url":      []byte(bootstrapAppRepositoryURL),
			"username": []byte("not-used"),
			"password": []byte("current-token"),
		},
	}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient, initialK8sSecret)

	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(awsSecretJSON))}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	currentK8sSecret := &corev1.Secret{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, currentK8sSecret)
	require.NoError(t, err)
	assert.Equal(t, initialK8sSecret.Data["password"], currentK8sSecret.Data["password"])
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_K8sSecretExists_NeedsUpdate_TokenDiffers(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}

	awsSecretData := awsRepositorySecret{GithubToken: "new-token-from-aws"} // pragma: allowlist secret
	awsSecretJSON, _ := json.Marshal(awsSecretData)

	initialK8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: argoCdNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":   "acsfleetctl",
				"argocd.argoproj.io/secret-type": "repository",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"url":      []byte(bootstrapAppRepositoryURL),
			"username": []byte("not-used"),
			"password": []byte("old-stale-token"), // Old token
		},
	}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient, initialK8sSecret)

	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(awsSecretJSON))}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	updatedK8sSecret := &corev1.Secret{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, updatedK8sSecret)
	require.NoError(t, err)
	assert.Equal(t, "new-token-from-aws", updatedK8sSecret.StringData["password"])
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_K8sSecretExists_NeedsUpdate_URLDiffers(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}

	awsSecretData := awsRepositorySecret{GithubToken: "same-token"} // pragma: allowlist secret
	awsSecretJSON, _ := json.Marshal(awsSecretData)

	initialK8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace},
		Data: map[string][]byte{
			"url":      []byte("https://github.com/some/other-repo"), // Different URL
			"password": []byte("same-token"),
		},
	}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient, initialK8sSecret)

	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(awsSecretJSON))}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	updatedK8sSecret := &corev1.Secret{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, updatedK8sSecret)
	require.NoError(t, err)
	assert.Equal(t, bootstrapAppRepositoryURL, updatedK8sSecret.StringData["url"]) // Should be updated to the constant
	assert.Equal(t, "same-token", updatedK8sSecret.StringData["password"])
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestEnsureRepositorySecret_K8sSecretExists_OwnedByESO_NoUpdate(t *testing.T) {
	ctx := context.Background()
	mockAwsSMClient := &secretsManagerClientMock{}

	awsSecretData := awsRepositorySecret{GithubToken: "new-aws-token"} // pragma: allowlist secret
	awsSecretJSON, _ := json.Marshal(awsSecretData)

	initialK8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: argoCdNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "external-secrets.io/v1beta1",
					Kind:       "ExternalSecret",
					Name:       "some-eso-secret",
					UID:        types.UID("some-uid"),
				},
			},
		},
		Data: map[string][]byte{
			"url":      []byte(bootstrapAppRepositoryURL),
			"password": []byte("token-managed-by-eso"),
		},
	}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient, initialK8sSecret)

	// AWS fetch will happen, but update should be skipped
	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(awsSecretJSON))}, nil
	}

	err := installer.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	currentK8sSecret := &corev1.Secret{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, currentK8sSecret)
	require.NoError(t, err)
	// Assert that the secret was NOT updated because it's owned by ESO
	assert.Equal(t, "token-managed-by-eso", string(currentK8sSecret.Data["password"]))
	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

func TestInstall(t *testing.T) {
	ctx := context.Background()
	infraName := "my-cluster-abcdef"

	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: infraName},
	}
	mockAwsSMClient := &secretsManagerClientMock{}
	installer := newTestInstallerWithAWSMock(mockAwsSMClient, initialInfra)

	awsSecretData := awsRepositorySecret{GithubToken: "install-token"} // pragma: allowlist secret
	awsSecretJSON, _ := json.Marshal(awsSecretData)
	mockAwsSMClient.GetSecretValueFunc = func(ctxMoq context.Context, paramsMoq *secretsmanager.GetSecretValueInput, optFnsMoq ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
		return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(string(awsSecretJSON))}, nil
	}

	err := installer.install(ctx)
	require.NoError(t, err)

	opNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: operatorNamespace}, opNs)
	require.NoError(t, err, "operatorNamespace should have been created")
	require.NotNil(t, opNs.Labels, "operatorNamespace labels map should be initialized")
	assert.Equal(t, managedByArgoCdLabelValue, opNs.Labels[managedByArgoCdLabelKey], "operatorNamespace managed-by label incorrect")

	argoNs := &corev1.Namespace{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: argoCdNamespace}, argoNs)
	require.NoError(t, err, "argoCDNamespace should have been created")
	require.NotNil(t, argoNs.Labels, "argoCDNamespace labels map should be initialized")
	assert.Equal(t, managedByArgoCdLabelValue, argoNs.Labels[managedByArgoCdLabelKey], "argoCDNamespace managed-by label incorrect")

	sub := &operatorsv1alpha1.Subscription{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: operatorNamespace}, sub)
	require.NoError(t, err, "Subscription should have been created by install()")

	repoSecret := &corev1.Secret{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: argoCdNamespace}, repoSecret)
	require.NoError(t, err, "Repository secret should have been created by install()")
	assert.Equal(t, "install-token", repoSecret.StringData["password"])

	require.Len(t, mockAwsSMClient.GetSecretValueCalls(), 1)
}

// MockK8sClientForCreateFailure is a wrapper to simulate create failures.
type MockK8sClientForCreateFailure struct {
	ctrlClient.Client
	shouldCreateFail bool
	createError      error
}

func (m *MockK8sClientForCreateFailure) Create(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
	if app, ok := obj.(*argoCd.Application); ok && app.Name == bootstrapAppName && m.shouldCreateFail {
		return m.createError
	}
	return m.Client.Create(ctx, obj, opts...)
}

func TestProcessInfraName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with suffix", "mycluster-abc-123xyz", "mycluster-abc"},
		{"single segment after dash", "mycluster-xyz", "mycluster"},
		{"no dash", "mycluster", "mycluster"},
		{"empty string", "", ""},
		{"only dash", "-", ""},
		{"ends with dash", "mycluster-", "mycluster"},
		{"multiple dashes internal", "my-internal-cluster-suffix", "my-internal-cluster"},
		{"short name with dash", "a-b", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, trimSuffixIfExists(tt.input))
		})
	}
}

func TestEnsureBootstrapApplication_DoesNotExist_WithHelmValue(t *testing.T) {
	ctx := context.Background()
	infraName := "my-cluster-abcdef"

	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: infraName},
	}
	installer := newTestInstaller(initialInfra)

	err := installer.ensureBootstrapApplication(ctx)
	require.NoError(t, err)

	createdApp := &argoCd.Application{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppName, Namespace: argoCdNamespace}, createdApp)
	require.NoError(t, err, "Bootstrap application should have been created")

	assert.Equal(t, bootstrapAppName, createdApp.Name)
	assert.Equal(t, argoCdNamespace, createdApp.Namespace)
}

func TestEnsureBootstrapApplication_GetInfraFails(t *testing.T) {
	ctx := context.Background()
	installer := newTestInstaller() // No Infrastructure object pre-populated

	err := installer.ensureBootstrapApplication(ctx)
	require.Error(t, err)
	assert.True(t, apiErrors.IsNotFound(err))
}

func TestEnsureBootstrapApplication_InfraStatusEmpty(t *testing.T) {
	ctx := context.Background()
	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: ""},
	}
	installer := newTestInstaller(initialInfra)

	err := installer.ensureBootstrapApplication(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "infrastructure name is empty the in status of CR")
}

func TestEnsureBootstrapApplication_Exists(t *testing.T) {
	ctx := context.Background()
	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: "some-cluster-name-123"},
	}
	existingApp := &argoCd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppName,
			Namespace: argoCdNamespace,
			Labels:    map[string]string{"existing-label": "true"},
		},
		Spec: argoCd.ApplicationSpec{
			Project: "custom-project",
			Source: &argoCd.ApplicationSource{
				RepoURL: bootstrapAppRepositoryURL,
				Path:    "old/path",
			},
			Destination: argoCd.ApplicationDestination{Server: "https://kubernetes.default.svc", Namespace: "other-ns"},
		},
	}
	installer := newTestInstaller(initialInfra, existingApp)

	err := installer.ensureBootstrapApplication(ctx)
	require.NoError(t, err) // Should skip creation and not error

	currentApp := &argoCd.Application{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: bootstrapAppName, Namespace: argoCdNamespace}, currentApp)
	require.NoError(t, err)
	assert.Equal(t, bootstrapAppName, currentApp.Name)
	assert.Equal(t, argoCdNamespace, currentApp.Namespace)
	assert.Equal(t, "custom-project", currentApp.Spec.Project, "Existing app project should not change")
	assert.Equal(t, "old/path", currentApp.Spec.Source.Path, "Existing app source path should not change")
	assert.Equal(t, "true", currentApp.Labels["existing-label"], "Existing app labels should persist")
}

func TestEnsureBootstrapApplication_AppCreateFails(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = argoCd.AddToScheme(scheme)
	_ = configv1.AddToScheme(scheme)

	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: "create-fail-cluster-infra"},
	}
	expectedCreateError := errors.New("K8S API error: failed to create application")

	mockK8sCreateFailClient := &MockK8sClientForCreateFailure{
		Client:           fake.NewClientBuilder().WithScheme(scheme).WithObjects(initialInfra).Build(),
		shouldCreateFail: true,
		createError:      expectedCreateError,
	}
	installer := &operatorInstaller{k8sClient: mockK8sCreateFailClient}

	err := installer.ensureBootstrapApplication(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating bootstrap application")
	assert.ErrorIs(t, err, expectedCreateError)
}
