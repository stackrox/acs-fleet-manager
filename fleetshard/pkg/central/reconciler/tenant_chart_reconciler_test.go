package reconciler

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartValues(t *testing.T) {

	r := tenantChartReconciler{chart: resourcesChart}

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
			values, err := r.chartValues(tt.managedCentral)
			tt.assertFn(t, values, err)
		})
	}

}
