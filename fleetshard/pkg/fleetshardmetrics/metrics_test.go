package fleetshardmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricIncrements(t *testing.T) {
	const expectedIncrement = 1.0

	tt := []struct {
		metricName    string
		incrementFunc func()
	}{
		{
			metricName:    "total_k8s_requests",
			incrementFunc: IncrementK8sRequests,
		},
		{
			metricName:    "total_k8s_request_errors",
			incrementFunc: IncrementK8sRequestErrors,
		},
		{
			metricName:    "total_fleet_manager_requests",
			incrementFunc: IncrementFleetManagerRequests,
		},
		{
			metricName:    "total_fleet_manager_request_errors",
			incrementFunc: IncrementFleetManagerRequestErrors,
		},
	}

	for _, tc := range tt {
		t.Run(tc.metricName, func(t *testing.T) {
			// reinit metrics to make sure global state of counters is at 0
			initMetrics()
			tc.incrementFunc()

			metrics := serveMetrics(t)
			targetMetric, hasKey := metrics[metricsPrefix+tc.metricName]
			require.Truef(t, hasKey, "expected metrics to contain %s but it did not: %v", tc.metricName, metrics)

			// Test that the metrics value is 1 after calling the incrementFunc
			value := targetMetric.Metric[0].Counter.Value
			assert.Equalf(t, expectedIncrement, *value, "expected metric: %s to have value: %v", tc.metricName, expectedIncrement)
		})
	}
}
