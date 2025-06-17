package controllers

import (
	"context"
	"errors"
	"testing"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stackrox/acs-fleet-manager/fleetshard-operator/api/v1alpha1"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	managerNamespace = "rhacs"
)

// newTestReconciler is the original helper, retained for tests not needing AWS mock
func newTestReconciler(initK8sObjs ...ctrlClient.Object) *ReconcileGitopsInstallation {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = argoCd.AddToScheme(scheme)
	_ = operatorsv1alpha1.AddToScheme(scheme)
	_ = operatorsv1.AddToScheme(scheme)
	_ = configv1.AddToScheme(scheme)

	fakeK8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initK8sObjs...).Build()
	return &ReconcileGitopsInstallation{
		Client:          fakeK8sClient,
		SourceNamespace: managerNamespace,
	}
}

func TestEnsureNamespace_DoesNotExist(t *testing.T) {
	ctx := context.Background()
	expectedLabelKey := managedByArgoCdLabelKey
	expectedLabelValue := managedByArgoCdLabelValue
	testNsName := "new-ns-test"
	reconciler := newTestReconciler()

	err := reconciler.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	createdNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: testNsName}, createdNs)
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
	reconciler := newTestReconciler(initialNs)

	err := reconciler.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	updatedNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: testNsName}, updatedNs)
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
	reconciler := newTestReconciler(initialNs)

	err := reconciler.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	currentNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: testNsName}, currentNs)
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
	reconciler := newTestReconciler(initialNs)

	err := reconciler.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	updatedNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: testNsName}, updatedNs)
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
	reconciler := newTestReconciler(initialNs)

	err := reconciler.ensureNamespace(ctx, testNsName)
	require.NoError(t, err)

	updatedNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: testNsName}, updatedNs)
	require.NoError(t, err)
	require.NotNil(t, updatedNs.Labels)
	assert.Equal(t, expectedLabelValue, updatedNs.Labels[expectedLabelKey], "Managed-by label should be set")
	assert.Equal(t, otherLabelValue, updatedNs.Labels[otherLabelKey], "Other existing labels should be preserved")
	assert.Len(t, updatedNs.Labels, 2, "Should have two labels")
}

func TestEnsureSubscription_DoesNotExist(t *testing.T) {
	ctx := context.Background()
	reconciler := newTestReconciler()

	err := reconciler.ensureSubscription(ctx)
	require.NoError(t, err)

	createdSub := &operatorsv1alpha1.Subscription{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: GitopsOperatorNamespace}, createdSub)
	require.NoError(t, err, "Subscription should have been created")

	assert.Equal(t, operatorSubscriptionName, createdSub.Name)
	assert.Equal(t, GitopsOperatorNamespace, createdSub.Namespace)
	require.NotNil(t, createdSub.Spec, "Created subscription spec should not be nil")
}

func TestEnsureSubscription_Exists(t *testing.T) {
	ctx := context.Background()
	existingSub := newSubscription()
	existingSub.Spec.Channel = "stable"

	reconciler := newTestReconciler(existingSub)

	err := reconciler.ensureSubscription(ctx)
	require.NoError(t, err)

	currentSub := &operatorsv1alpha1.Subscription{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: GitopsOperatorNamespace}, currentSub)
	require.NoError(t, err, "Subscription should still exist")

	// The current ensureSubscription logic does not update an existing subscription.
	// It only checks for existence and creates if not found.
	assert.Equal(t, existingSub.Spec.Channel, currentSub.Spec.Channel, "Existing subscription channel should not have changed")
	assert.Equal(t, existingSub.Spec.Package, currentSub.Spec.Package)
}

func TestEnsureRepositorySecret_SourceNotFound(t *testing.T) {
	ctx := context.Background()

	reconciler := newTestReconciler() // no source secret is created.

	err := reconciler.ensureRepositorySecret(ctx)
	require.Error(t, err)
	assert.True(t, apiErrors.IsNotFound(err), "Error should be IsNotFound")
	assert.Contains(t, err.Error(), "source secret 'acscs-manifests-repo' in namespace 'rhacs' not found")
}

func TestEnsureRepositorySecret_SourceSecretMissingKey(t *testing.T) {
	ctx := context.Background()

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: managerNamespace,
		},
		Data: map[string][]byte{
			"some-other-key": []byte("some-value"), // Does not contain the required 'github-token' key
		},
	}
	reconciler := newTestReconciler(sourceSecret)

	err := reconciler.ensureRepositorySecret(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not contain a non-empty key 'github-token'")
}

func TestEnsureRepositorySecret_DestinationDoesNotExist(t *testing.T) {
	ctx := context.Background()
	token := "my-secret-git-token"

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapAppRepositoryName,
			Namespace: managerNamespace,
		},
		Data: map[string][]byte{
			tokenKey: []byte(token),
		},
	}
	reconciler := newTestReconciler(sourceSecret)

	err := reconciler.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	// Verify destination secret was created
	destSecret := &corev1.Secret{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace}, destSecret)
	require.NoError(t, err, "Destination secret should have been created")

	// Verify content
	assert.Equal(t, bootstrapAppRepositoryURL, string(destSecret.Data["url"]))
	assert.Equal(t, token, string(destSecret.Data["password"]))
	assert.Equal(t, "repository", destSecret.Labels["argocd.argoproj.io/secret-type"])
}

func TestEnsureRepositorySecret_DestinationExists_NeedsUpdate(t *testing.T) {
	ctx := context.Background()
	newToken := "new-shiny-token"
	oldToken := "old-stale-token"

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: bootstrapAppRepositoryName, Namespace: managerNamespace},
		Data:       map[string][]byte{tokenKey: []byte(newToken)},
	}

	// Destination secret exists with an old token
	destSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace},
		Data:       map[string][]byte{"password": []byte(oldToken), "url": []byte(bootstrapAppRepositoryURL)},
	}
	reconciler := newTestReconciler(sourceSecret, destSecret)

	err := reconciler.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	// Verify destination secret was updated
	updatedDestSecret := &corev1.Secret{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace}, updatedDestSecret)
	require.NoError(t, err)
	assert.Equal(t, newToken, string(updatedDestSecret.Data["password"]))
}

func TestEnsureRepositorySecret_DestinationExists_IsUpToDate(t *testing.T) {
	ctx := context.Background()
	token := []byte("current-token")

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: bootstrapAppRepositoryName, Namespace: managerNamespace},
		Data:       map[string][]byte{tokenKey: token},
	}

	// Destination secret already exists and is identical to what would be created
	destSecret := newRepositorySecret(token)

	installer := newTestReconciler(sourceSecret, destSecret)

	initialResourceVersion := destSecret.ResourceVersion

	err := installer.ensureRepositorySecret(ctx)
	require.NoError(t, err)

	// Verify destination secret was not modified
	finalDestSecret := &corev1.Secret{}
	err = installer.Client.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace}, finalDestSecret)
	require.NoError(t, err)
	assert.Equal(t, token, finalDestSecret.Data["password"])
	assert.Equal(t, initialResourceVersion, finalDestSecret.ResourceVersion, "ResourceVersion should not change if secret is up-to-date")
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	initialGitopsInstallation := &v1alpha1.GitopsInstallation{
		ObjectMeta: metav1.ObjectMeta{Namespace: managerNamespace, Name: "rhacs-gitops"},
		Spec: v1alpha1.GitopsInstallationSpec{
			ClusterName:                "my-cluster",
			BootstrapAppTargetRevision: "HEAD",
		},
	}
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: bootstrapAppRepositoryName, Namespace: managerNamespace},
		Data:       map[string][]byte{tokenKey: []byte("install-token")},
	}
	reconciler := newTestReconciler(initialGitopsInstallation, sourceSecret)

	_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "rhacs-gitops", Namespace: managerNamespace}})
	require.NoError(t, err)

	opNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: GitopsOperatorNamespace}, opNs)
	require.NoError(t, err, "GitopsOperatorNamespace should have been created")
	require.NotNil(t, opNs.Labels, "GitopsOperatorNamespace labels map should be initialized")
	assert.Equal(t, managedByArgoCdLabelValue, opNs.Labels[managedByArgoCdLabelKey], "GitopsOperatorNamespace managed-by label incorrect")

	argoNs := &corev1.Namespace{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: ArgoCdNamespace}, argoNs)
	require.NoError(t, err, "argoCDNamespace should have been created")
	require.NotNil(t, argoNs.Labels, "argoCDNamespace labels map should be initialized")
	assert.Equal(t, managedByArgoCdLabelValue, argoNs.Labels[managedByArgoCdLabelKey], "argoCDNamespace managed-by label incorrect")

	sub := &operatorsv1alpha1.Subscription{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: GitopsOperatorNamespace}, sub)
	require.NoError(t, err, "Subscription should have been created by Reconcile()")

	repoSecret := &corev1.Secret{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: bootstrapAppRepositoryName, Namespace: ArgoCdNamespace}, repoSecret)
	require.NoError(t, err, "Repository secret should have been created by Reconcile()")
	assert.Equal(t, "install-token", string(repoSecret.Data["password"]))
}

func TestCreateInitialInstallation(t *testing.T) {
	ctx := context.Background()

	infraName := "my-cluster-abcdef"
	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: infraName},
	}
	reconciler := newTestReconciler(initialInfra)
	err := reconciler.createDefaultGitopsInstallation(ctx)
	require.NoError(t, err)
	installation := &v1alpha1.GitopsInstallation{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: "rhacs-gitops", Namespace: managerNamespace}, installation)
	require.NoError(t, err, "GitopsInstallation should have been created")
	require.Equal(t, "my-cluster", installation.Spec.ClusterName)
	require.Equal(t, "HEAD", installation.Spec.BootstrapAppTargetRevision)
}

func TestCreateInitialInstallation_AlreadyExists(t *testing.T) {
	ctx := context.Background()

	infraName := "my-cluster-abcdef"
	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: infraName},
	}
	installation := &v1alpha1.GitopsInstallation{
		ObjectMeta: metav1.ObjectMeta{Namespace: managerNamespace, Name: "rhacs-gitops"},
		Spec: v1alpha1.GitopsInstallationSpec{
			ClusterName:                "my-cluster",
			BootstrapAppTargetRevision: "HEAD",
		},
	}
	reconciler := newTestReconciler(initialInfra, installation)
	err := reconciler.createDefaultGitopsInstallation(ctx)
	require.NoError(t, err)
}

func TestCreateInitialInstallation_GetInfraFails(t *testing.T) {
	ctx := context.Background()

	reconciler := newTestReconciler() // no infrastructure object populated
	err := reconciler.createDefaultGitopsInstallation(ctx)
	require.Error(t, err)
	assert.True(t, apiErrors.IsNotFound(err))
}

func TestCreateInitialInstallation_InfraStatusEmpty(t *testing.T) {
	ctx := context.Background()
	initialInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Status:     configv1.InfrastructureStatus{InfrastructureName: ""},
	}
	reconciler := newTestReconciler(initialInfra)

	err := reconciler.createDefaultGitopsInstallation(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "infrastructure name is empty the in status of CR")
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
	reconciler := newTestReconciler(initialInfra)

	err := reconciler.ensureBootstrapApplication(ctx, v1alpha1.GitopsInstallationSpec{})
	require.NoError(t, err)

	createdApp := &argoCd.Application{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: bootstrapAppName, Namespace: ArgoCdNamespace}, createdApp)
	require.NoError(t, err, "Bootstrap application should have been created")

	assert.Equal(t, bootstrapAppName, createdApp.Name)
	assert.Equal(t, ArgoCdNamespace, createdApp.Namespace)
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
			Namespace: ArgoCdNamespace,
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
	reconciler := newTestReconciler(initialInfra, existingApp)

	err := reconciler.ensureBootstrapApplication(ctx, v1alpha1.GitopsInstallationSpec{})
	require.NoError(t, err) // Should skip creation and not error

	currentApp := &argoCd.Application{}
	err = reconciler.Client.Get(ctx, types.NamespacedName{Name: bootstrapAppName, Namespace: ArgoCdNamespace}, currentApp)
	require.NoError(t, err)
	assert.Equal(t, bootstrapAppName, currentApp.Name)
	assert.Equal(t, ArgoCdNamespace, currentApp.Namespace)
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
	reconciler := &ReconcileGitopsInstallation{Client: mockK8sCreateFailClient}

	err := reconciler.ensureBootstrapApplication(ctx, v1alpha1.GitopsInstallationSpec{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating bootstrap application")
	assert.ErrorIs(t, err, expectedCreateError)
}
