package reconciler

import (
	"context"
	"testing"

	"github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDisablePauseAnnotation(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	central.Annotations[PauseReconcileAnnotation] = "true"
	err = fakeClient.Update(context.TODO(), central)
	require.NoError(t, err)

	err = r.centralCrReconciler.disablePauseReconcileIfPresent(context.TODO(), central)
	require.NoError(t, err)

	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	require.Equal(t, "false", central.Annotations[PauseReconcileAnnotation])
}
