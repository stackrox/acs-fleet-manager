package reconciler

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_tenantResourcesChartReconciler_createsResources(t *testing.T) {
	fakeChart, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)
	chart, err := loader.LoadFiles(fakeChart)
	require.NoError(t, err)

	fakeClient := fake2.NewFakeClient()

	r := newTenantResourcesChartReconciler(fakeClient, chart, false)

	ctx := context.Background()
	ctx = withManagedCentral(ctx, private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Namespace: "test-namespace",
		},
	})

	_, err = r.ensurePresent(ctx)
	require.NoError(t, err)

	var dummyObj networkingv1.NetworkPolicy
	dummyObjKey := client.ObjectKey{Namespace: "test-namespace", Name: "dummy"}
	err = fakeClient.Get(ctx, dummyObjKey, &dummyObj)
	assert.NoError(t, err)

	assert.Equal(t, k8s.ManagedByFleetshardValue, dummyObj.GetLabels()[k8s.ManagedByLabelKey])

	_, err = r.ensureAbsent(ctx)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, dummyObjKey, &dummyObj)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func Test_tenantResourcesChartReconciler_updatesResources(t *testing.T) {
	chartFiles, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)
	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)

	fakeClient := fake2.NewFakeClient()

	r := newTenantResourcesChartReconciler(fakeClient, chart, false)

	ctx := context.Background()
	ctx = withManagedCentral(ctx, private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Namespace: "test-namespace",
		},
	})

	_, err = r.ensurePresent(ctx)
	require.NoError(t, err)

	var dummyObj networkingv1.NetworkPolicy
	dummyObjKey := client.ObjectKey{Namespace: "test-namespace", Name: "dummy"}
	err = fakeClient.Get(ctx, dummyObjKey, &dummyObj)
	assert.NoError(t, err)

	dummyObj.SetAnnotations(map[string]string{"dummy-annotation": "test"})
	err = fakeClient.Update(ctx, &dummyObj)
	assert.NoError(t, err)

	err = fakeClient.Get(ctx, dummyObjKey, &dummyObj)
	assert.NoError(t, err)
	assert.Equal(t, "test", dummyObj.GetAnnotations()["dummy-annotation"])

	_, err = r.ensurePresent(ctx)
	require.NoError(t, err)

	err = fakeClient.Get(ctx, dummyObjKey, &dummyObj)
	assert.NoError(t, err)

	// verify that the chart resource was updated, by checking that the manually added annotation
	// is no longer present
	assert.Equal(t, "", dummyObj.GetAnnotations()["dummy-annotation"])
}

func Test_tenantResourcesChartReconciler_chartValues(t *testing.T) {

	chartFiles, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)

	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)

	tests := []struct {
		name           string
		managedCentral private.ManagedCentral
		assertFn       func(t *testing.T, values map[string]interface{}, err error)
	}{
		{
			name: "withTenantResources",
			managedCentral: private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					TenantResourcesValues: map[string]interface{}{
						"verticalPodAutoscalers": map[string]interface{}{
							"central": map[string]interface{}{
								"enabled": true,
								"updatePolicy": map[string]interface{}{
									"minReplicas": 1,
								},
							},
						},
					},
				},
			},
			assertFn: func(t *testing.T, values map[string]interface{}, err error) {

				require.NoError(t, err)
				verticalPodAutoscalers, ok := values["verticalPodAutoscalers"].(map[string]interface{})
				require.True(t, ok)
				central, ok := verticalPodAutoscalers["central"].(map[string]interface{})
				require.True(t, ok)
				enabled, ok := central["enabled"].(bool)
				require.True(t, ok)
				assert.True(t, enabled)

				updatePolicy, ok := central["updatePolicy"].(map[string]interface{})
				require.True(t, ok)
				minReplicas, ok := updatePolicy["minReplicas"].(int)
				require.True(t, ok)
				assert.Equal(t, 1, minReplicas)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tenantResourcesChartReconciler{resourcesChart: chart}
			values, err := r.chartValues(tt.managedCentral)
			tt.assertFn(t, values, err)
		})
	}

}
