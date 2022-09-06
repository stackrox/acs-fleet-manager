package fleetshardmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricIncrements(t *testing.T) {
	const expectedIncrement = 1.0

	tt := []struct {
		metricName        string
		callIncrementFunc func(m *Metrics)
	}{
		{
			metricName: "total_fleet_manager_requests",
			callIncrementFunc: func(m *Metrics) {
				m.IncrementFleetManagerRequests()
			},
		},
		{
			metricName: "total_fleet_manager_request_errors",
			callIncrementFunc: func(m *Metrics) {
				m.IncrementFleetManagerRequestErrors()
			},
		},
		{
			metricName: "total_central_reconcilations",
			callIncrementFunc: func(m *Metrics) {
				m.IncrementCentralReconcilations()
			},
		},
		{
			metricName: "total_central_reconcilation_errors",
			callIncrementFunc: func(m *Metrics) {
				m.IncrementCentralReconcilationErrors()
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			tc.callIncrementFunc(m)

			metrics := serveMetrics(t, m)
			targetMetric, hasKey := metrics[metricsPrefix+tc.metricName]
			require.Truef(t, hasKey, "expected metrics to contain %s but it did not: %v", tc.metricName, metrics)

			// Test that the metrics value is 1 after calling the incrementFunc
			value := targetMetric.Metric[0].Counter.Value
			assert.Equalf(t, expectedIncrement, *value, "expected metric: %s to have value: %v", tc.metricName, expectedIncrement)
		})
	}
}
