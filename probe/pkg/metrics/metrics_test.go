package metrics

import (
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
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
			metricName: "acs_probe_runs_started",
			callIncrementFunc: func(m *Metrics) {
				m.IncStartedRuns()
			},
		},
		{
			metricName: "acs_probe_runs_success",
			callIncrementFunc: func(m *Metrics) {
				m.IncSuccessfulRuns()
			},
		},
		{
			metricName: "acs_probe_runs_failed",
			callIncrementFunc: func(m *Metrics) {
				m.IncFailedRuns()
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			tc.callIncrementFunc(m)

			metrics := serveMetrics(t, m)
			targetMetric := requireMetric(t, metrics, tc.metricName)

			// Test that the metrics value is 1 after calling the incrementFunc.
			value := targetMetric.Metric[0].GetCounter().GetValue()
			assert.Equalf(t, expectedIncrement, value, "metric %s has unexpected value", tc.metricName)
		})
	}
}

func TestTimestampGauges(t *testing.T) {
	tt := []struct {
		metricName       string
		setTimestampFunc func(m *Metrics)
	}{
		{
			metricName: "acs_probe_last_success_timestamp",
			setTimestampFunc: func(m *Metrics) {
				m.SetLastSuccessTimestamp()
			},
		},
		{
			metricName: "acs_probe_last_failed_timestamp",
			setTimestampFunc: func(m *Metrics) {
				m.SetLastFailureTimestamp()
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			lowerBound := time.Now().Unix()
			tc.setTimestampFunc(m)

			metrics := serveMetrics(t, m)
			targetMetric := requireMetric(t, metrics, tc.metricName)

			value := int64(targetMetric.Metric[0].GetGauge().GetValue())
			assert.GreaterOrEqualf(t, value, lowerBound, "metric %s has unexpected value", tc.metricName)
		})
	}
}

func TestHistograms(t *testing.T) {
	tt := []struct {
		metricName  string
		observeFunc func(m *Metrics)
	}{
		{
			metricName: "acs_probe_total_duration_seconds",
			observeFunc: func(m *Metrics) {
				m.ObserveTotalDuration(5 * time.Minute)
				m.ObserveTotalDuration(3 * time.Minute)
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.metricName, func(t *testing.T) {
			m := newMetrics()
			expectedCount := uint64(2)
			expectedSum := 480.0
			tc.observeFunc(m)

			metrics := serveMetrics(t, m)
			targetMetric := requireMetric(t, metrics, tc.metricName)

			count := targetMetric.Metric[0].GetHistogram().GetSampleCount()
			sum := targetMetric.Metric[0].GetHistogram().GetSampleSum()
			assert.Equalf(t, expectedCount, count, "expected metric: %s to have a count of %v", tc.metricName, expectedCount)
			assert.Equalf(t, expectedSum, sum, "expected metric: %s to have a sum of %v", tc.metricName, expectedSum)
		})
	}
}

func requireMetric(t *testing.T, metrics metricResponse, metricName string) *io_prometheus_client.MetricFamily {
	targetMetric := metrics[metricName]
	require.NotNilf(t, targetMetric, "expected metrics to contain %s but it did not: %v", metricName, metrics)
	return targetMetric
}
