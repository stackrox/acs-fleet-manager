package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	regionValue = "us-east-1"
	cfg         = config.Config{
		MetricsAddress: ":8081",
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
			metricName: "acs_probe_runs_started_total",
			callIncrementFunc: func(m *Metrics) {
				m.IncStartedRuns(regionValue)
			},
		},
		{
			metricName: "acs_probe_runs_succeeded_total",
			callIncrementFunc: func(m *Metrics) {
				m.IncSucceededRuns(regionValue)
			},
		},
		{
			metricName: "acs_probe_runs_failed_total",
			callIncrementFunc: func(m *Metrics) {
				m.IncFailedRuns(regionValue)
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
			assert.Containsf(t, label.GetName(), regionLabelName, "metric %s has unexpected label", tc.metricName)
			assert.Containsf(t, label.GetValue(), regionValue, "metric %s has unexpected label", tc.metricName)
		})
	}
}

func TestTimestampGauges(t *testing.T) {
	tt := []struct {
		metricName           string
		callSetTimestampFunc func(m *Metrics)
	}{
		{
			metricName: "acs_probe_last_started_timestamp",
			callSetTimestampFunc: func(m *Metrics) {
				m.SetLastStartedTimestamp(regionValue)
			},
		},
		{
			metricName: "acs_probe_last_success_timestamp",
			callSetTimestampFunc: func(m *Metrics) {
				m.SetLastSuccessTimestamp(regionValue)
			},
		},
		{
			metricName: "acs_probe_last_failure_timestamp",
			callSetTimestampFunc: func(m *Metrics) {
				m.SetLastFailureTimestamp(regionValue)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			registry := initPrometheus(m)
			lowerBound := time.Now().Unix()

			tc.callSetTimestampFunc(m)

			targetSeries := getMetricSeries(t, registry, tc.metricName)
			value := int64(targetSeries.GetGauge().GetValue())
			assert.GreaterOrEqualf(t, value, lowerBound, "metric %s has unexpected value", tc.metricName)
			label := targetSeries.GetLabel()[0]
			assert.Containsf(t, label.GetName(), regionLabelName, "metric %s has unexpected label", tc.metricName)
			assert.Containsf(t, label.GetValue(), regionValue, "metric %s has unexpected label", tc.metricName)
		})
	}
}

func TestHistograms(t *testing.T) {
	tt := []struct {
		metricName      string
		callObserveFunc func(m *Metrics)
	}{
		{
			metricName: "acs_probe_total_duration_seconds",
			callObserveFunc: func(m *Metrics) {
				m.ObserveTotalDuration(5*time.Minute, regionValue)
				m.ObserveTotalDuration(3*time.Minute, regionValue)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			registry := initPrometheus(m)
			expectedCount := uint64(2)
			expectedSum := 480.0
			tc.callObserveFunc(m)

			targetSeries := getMetricSeries(t, registry, tc.metricName)

			count := targetSeries.GetHistogram().GetSampleCount()
			sum := targetSeries.GetHistogram().GetSampleSum()
			assert.Equalf(t, expectedCount, count, "expected metric: %s to have a count of %v", tc.metricName, expectedCount)
			assert.Equalf(t, expectedSum, sum, "expected metric: %s to have a sum of %v", tc.metricName, expectedSum)
			label := targetSeries.GetLabel()[0]
			assert.Containsf(t, label.GetName(), regionLabelName, "metric %s has unexpected label", tc.metricName)
			assert.Containsf(t, label.GetValue(), regionValue, "metric %s has unexpected label", tc.metricName)
		})
	}
}

func TestMetricsConformity(t *testing.T) {
	metrics := newMetrics()

	for _, metric := range []prometheus.Collector{
		metrics.startedRuns, metrics.succeededRuns, metrics.failedRuns, metrics.lastStartedTimestamp,
		metrics.lastSuccessTimestamp, metrics.lastFailureTimestamp, metrics.totalDurationHistogram,
	} {
		problems, err := testutil.CollectAndLint(metric)
		assert.NoError(t, err)
		assert.Empty(t, problems)
	}
}
