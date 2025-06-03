package gitops

import (
	"context"
	"testing"

	argoCd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestInstaller(initObjs ...ctrlClient.Object) *operatorInstaller {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = argoCd.AddToScheme(scheme)
	_ = operatorsv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build()
	return &operatorInstaller{
		k8sClient: fakeClient,
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
	existingSub := newSubscription()    // Use the helper to create a valid existing subscription
	existingSub.Spec.Channel = "stable" // Modify slightly to see if it's overwritten (it shouldn't be by current logic)

	installer := newTestInstaller(existingSub)

	err := installer.ensureSubscription(ctx)
	require.NoError(t, err)

	// Verify Subscription still exists and was not modified
	currentSub := &operatorsv1alpha1.Subscription{}
	err = installer.k8sClient.Get(ctx, types.NamespacedName{Name: operatorSubscriptionName, Namespace: operatorNamespace}, currentSub)
	require.NoError(t, err, "Subscription should still exist")

	// The current ensureSubscription logic does not update an existing subscription.
	// It only checks for existence and creates if not found.
	assert.Equal(t, existingSub.Spec.Channel, currentSub.Spec.Channel, "Existing subscription channel should not have changed")
	assert.Equal(t, existingSub.Spec.Package, currentSub.Spec.Package)
}

func TestInstall(t *testing.T) {
	ctx := context.Background()
	installer := newTestInstaller()

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
}
