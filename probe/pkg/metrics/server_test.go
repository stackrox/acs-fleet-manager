package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metricResponse map[string]*io_prometheus_client.MetricFamily

func TestMetricsServerCorrectAddress(t *testing.T) {
	server := NewMetricsServer(cfg)
	defer server.Close()
	assert.Equal(t, ":8081", server.Addr)
}

func TestMetricsServerServesDefaultMetrics(t *testing.T) {
	registry := initPrometheus(newMetrics())
	metrics := serveMetrics(t, registry)
	assert.Contains(t, metrics, "go_memstats_alloc_bytes", "expected metrics to contain go default metrics but it did not")
}

func TestMetricsServerServesCustomMetrics(t *testing.T) {
	// Initialize metric series such that they are not empty.
	customMetrics := newMetrics()
	customMetrics.IncStartedRuns(regionValue)
	customMetrics.IncSucceededRuns(regionValue)
	customMetrics.IncFailedRuns(regionValue)
	customMetrics.SetLastStartedTimestamp(regionValue)
	customMetrics.SetLastSuccessTimestamp(regionValue)
	customMetrics.SetLastFailureTimestamp(regionValue)
	customMetrics.ObserveTotalDuration(time.Minute, regionValue)
	registry := initPrometheus(customMetrics)
	metrics := serveMetrics(t, registry)

	expectedKeys := []string{
		"acs_probe_runs_started_total",
		"acs_probe_runs_succeeded_total",
		"acs_probe_runs_failed_total",
		"acs_probe_last_started_timestamp",
		"acs_probe_last_success_timestamp",
		"acs_probe_last_failure_timestamp",
		"acs_probe_total_duration_seconds",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, metrics, key, "custom metric not contained in metrics")
	}
}

func serveMetrics(t *testing.T, registry *prometheus.Registry) metricResponse {
	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err, "failed creating metrics requests")

	server := newMetricsServer(":8081", registry)
	defer server.Close()
	server.Handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

	promParser := expfmt.NewTextParser(model.UTF8Validation)
	metrics, err := promParser.TextToMetricFamilies(rec.Body)
	require.NoError(t, err, "failed parsing metrics response")
	return metrics
}
