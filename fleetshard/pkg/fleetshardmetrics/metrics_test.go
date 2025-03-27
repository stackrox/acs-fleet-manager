package fleetshardmetrics

import (
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterIncrements(t *testing.T) {
	const expectedIncrement = 1.0

	tt := []struct {
		metricName        string
		callIncrementFunc func(m *Metrics)
	}{
		{
			metricName: "total_central_reconcilations",
			callIncrementFunc: func(m *Metrics) {
				m.IncCentralReconcilations()
			},
		},
		{
			metricName: "total_central_reconcilation_errors",
			callIncrementFunc: func(m *Metrics) {
				m.AddCentralReconcilationErrors(1)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			tc.callIncrementFunc(m)

			metrics := serveMetrics(t, m)
			targetMetric := requireMetric(t, metrics, metricsPrefix+tc.metricName)

			// Test that the metrics value is 1 after calling the incrementFunc
			value := targetMetric.Metric[0].Counter.Value
			assert.Equalf(t, expectedIncrement, *value, "expected metric: %s to have value: %v", tc.metricName, expectedIncrement)
		})
	}
}

func TestTotalCentrals(t *testing.T) {
	m := newMetrics()
	metricName := metricsPrefix + "total_centrals"
	expectedValue := 37

	m.SetTotalCentrals(expectedValue)
	metrics := serveMetrics(t, m)

	targetMetric := requireMetric(t, metrics, metricName)
	value := targetMetric.Metric[0].Gauge.Value
	assert.Equalf(t, 37.0, *value, "expected metric: %s to have value: %v", metricName, expectedValue)
}

func TestReadyCentrals(t *testing.T) {
	m := newMetrics()
	metricName := metricsPrefix + "ready_centrals"
	expectedValue := 37

	m.SetReadyCentrals(expectedValue)
	metrics := serveMetrics(t, m)

	targetMetric := requireMetric(t, metrics, metricName)
	value := targetMetric.Metric[0].Gauge.Value
	assert.Equalf(t, 37.0, *value, "expected metric: %s to have value: %v", metricName, expectedValue)
}

func TestActiveCentralReconcilations(t *testing.T) {
	m := newMetrics()
	metricName := metricsPrefix + "active_central_reconcilations"

	m.IncActiveCentralReconcilations()
	metrics := serveMetrics(t, m)

	targetMetric := requireMetric(t, metrics, metricName)
	value := targetMetric.Metric[0].Gauge.Value
	assert.Equalf(t, 1.0, *value, "expected metric: %s to have value: %v", metricName, 1.0)

	m.DecActiveCentralReconcilations()
	metrics = serveMetrics(t, m)

	targetMetric = requireMetric(t, metrics, metricName)
	value = targetMetric.Metric[0].Gauge.Value
	assert.Equalf(t, 0.0, *value, "expected metric: %s to have value: %v", metricName, 0.0)
}

func TestDatabaseQuotaMetrics(t *testing.T) {
	m := newMetrics()
	m.SetDatabaseAccountQuotas(cloudprovider.AccountQuotas{
		cloudprovider.DBClusters:  {Used: 2, Max: 40},
		cloudprovider.DBInstances: {Used: 4, Max: 100},
		cloudprovider.DBSnapshots: {Used: 15, Max: 700},
	})

	metrics := serveMetrics(t, m)

	expectedValues := map[string]float64{
		"central_db_clusters_used":  2.0,
		"central_db_clusters_max":   40.0,
		"central_db_instances_used": 4.0,
		"central_db_instances_max":  100.0,
		"central_db_snapshots_used": 15.0,
		"central_db_snapshots_max":  700.0,
	}

	for key, expectedValue := range expectedValues {
		metricName := metricsPrefix + key
		targetMetric := requireMetric(t, metrics, metricName)
		value := targetMetric.Metric[0].Gauge.Value
		assert.Equalf(t, expectedValue, *value, "expected metric: %s to have value: %v", metricName, expectedValue)
	}
}

func TestPauseReconcileMetrics(t *testing.T) {
	m := newMetrics()

	m.SetPauseReconcileStatus("instance-1", false)
	m.SetPauseReconcileStatus("instance-2", true)

	metrics := serveMetrics(t, m)

	expectedValues := map[string]float64{
		"instance-1": 0.0,
		"instance-2": 1.0,
	}

	metricName := metricsPrefix + "pause_reconcile_instances"
	targetMetric := requireMetric(t, metrics, metricName)
	require.Equal(t, 2, len(targetMetric.Metric))

	for _, metric := range targetMetric.Metric {
		require.Equal(t, 1, len(metric.Label))
		require.Equal(t, "instance", *metric.Label[0].Name)
		instanceName := *metric.Label[0].Value

		assert.Equal(t, expectedValues[instanceName], *metric.Gauge.Value)
	}
}

func requireMetric(t *testing.T, metrics metricResponse, metricName string) *io_prometheus_client.MetricFamily {
	targetMetric, hasKey := metrics[metricName]
	require.Truef(t, hasKey, "expected metrics to contain %s but it did not: %v", metricName, metrics)
	return targetMetric
}
