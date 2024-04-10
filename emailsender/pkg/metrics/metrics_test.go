package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	clusterID = "test-1"
	cfg       = &config.Config{
		ClusterID:      clusterID,
		MetricsAddress: ":9999",
	}
)

func getMetricSeries(t *testing.T, registry *prometheus.Registry, name string) *io_prometheus_client.Metric {
	metrics := serveMetrics(t, registry)
	require.Contains(t, metrics, name)
	targetMetric := metrics[name]
	require.NotEmpty(t, targetMetric.Metric)
	return targetMetric.Metric[0]
}

func TestCounterIncrements(t *testing.T) {
	const expectedIncrement = 1.0

	tt := []struct {
		metricName        string
		callIncrementFunc func(m *Metrics)
	}{
		{
			metricName: "acs_emailsender_email_sent_total",
			callIncrementFunc: func(m *Metrics) {
				m.IncEmailsSent(cfg.ClusterID)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			registry := initPrometheus(m)
			tc.callIncrementFunc(m)

			targetSeries := getMetricSeries(t, registry, tc.metricName)

			// Test that the metrics value is 1 after calling the incrementFunc.
			value := targetSeries.GetCounter().GetValue()
			assert.Equalf(t, expectedIncrement, value, "metric %s has unexpected value", tc.metricName)
			label := targetSeries.GetLabel()[0]
			assert.Containsf(t, label.GetName(), clusterIDLabelName, "metric %s has unexpected label", tc.metricName)
			assert.Containsf(t, label.GetValue(), clusterID, "metric %s has unexpected label", tc.metricName)
		})
	}
}

func TestMetricsConformity(t *testing.T) {
	metrics := newMetrics()

	for _, metric := range []prometheus.Collector{
		metrics.emailsSent,
	} {
		problems, err := testutil.CollectAndLint(metric)
		assert.NoError(t, err)
		assert.Empty(t, problems)
	}
}
