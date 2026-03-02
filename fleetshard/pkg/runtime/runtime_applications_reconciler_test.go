package runtime

import (
	"context"
	"testing"

	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
)

func Test_runtimeApplicationsReconciler_reconcile_emptyList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().WithScheme(scheme).Build()
	err := newRuntimeApplicationsReconciler(cli, "openshift-gitops").reconcile(context.Background(), private.ManagedCentralList{
		Applications: []map[string]interface{}{},
	})

	assert.NoError(t, err)
}

func Test_runtimeApplicationsReconciler_reconcile_nonEmptyList(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	cli := fake.NewClientBuilder().WithScheme(scheme).Build()
	err := newRuntimeApplicationsReconciler(cli, "openshift-gitops").reconcile(context.Background(), private.ManagedCentralList{
		Applications: []map[string]interface{}{
			{
				"metadata": map[string]interface{}{
					"name": "app-1",
				},
			},
		},
	})
	assert.NoError(t, err)

	var appList argocd.ApplicationList
	require.NoError(t, cli.List(context.Background(), &appList))
	require.Len(t, appList.Items, 1)
	assert.Equal(t, "app-1", appList.Items[0].Name)
}

func Test_runtimeApplicationsReconciler_reconcile_shouldNotUpdateForSameValue(t *testing.T) {

	scheme := runtime.NewScheme()
	require.NoError(t, argocd.AddToScheme(scheme))

	updateCount := 0
	createCount := 0

	cli := fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
		Create: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
			createCount++
			return client.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.UpdateOption) error {
			updateCount++
			return client.Update(ctx, obj, opts...)
		},
	}).WithScheme(scheme).Build()

	reconciler := newRuntimeApplicationsReconciler(cli, "openshift-gitops")

	input := private.ManagedCentralList{
		Applications: []map[string]interface{}{
			{
				"metadata": map[string]interface{}{
					"name": "app-1",
				},
			},
		},
	}

	err := reconciler.reconcile(context.Background(), input)
	require.NoError(t, err)
	err = reconciler.reconcile(context.Background(), input)
	require.NoError(t, err)

	assert.Equal(t, 1, createCount)
	assert.Equal(t, 0, updateCount)

	// Change something in the input
	input = private.ManagedCentralList{
		Applications: []map[string]interface{}{
			{
				"metadata": map[string]interface{}{
					"name": "app-1",
					"annotations": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
		},
	}

	err = reconciler.reconcile(context.Background(), input)
	require.NoError(t, err)

	assert.Equal(t, 1, createCount)
	assert.Equal(t, 1, updateCount)

}
