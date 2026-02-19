package fleetshardmetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metricResponse map[string]*io_prometheus_client.MetricFamily

func TestMetricsServerCorrectAddress(t *testing.T) {
	server := NewMetricsServer(":8081")
	assert.Equal(t, ":8081", server.Addr)
}

func TestMetricsServerServesDefaultMetrics(t *testing.T) {
	metrics := serveMetrics(t, newMetrics())
	_, hasKey := metrics["go_memstats_alloc_bytes"]
	assert.Truef(t, hasKey, "expected metrics to contain go default metrics but it did not: %v", metrics)
}

func TestMetricsServerServesCustomMetrics(t *testing.T) {
	metrics := serveMetrics(t, newMetrics())

	expectedKeys := []string{
		"total_fleet_manager_requests",
		"total_fleet_manager_request_errors",
		"total_central_reconcilations",
		"total_central_reconcilation_errors",
		"active_central_reconcilations",
		"total_centrals",
		"ready_centrals",
	}

	for _, key := range expectedKeys {
		assert.Containsf(t, metrics, metricsPrefix+key, "expected metrics to contain %s but it did not: %v", key, metrics)
	}
}

func serveMetrics(t *testing.T, customMetrics *Metrics) metricResponse {
	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err, "failed creating metrics requests")

	server := newMetricsServer(":8081", customMetrics)
	server.Handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

	promParser := expfmt.NewTextParser(model.UTF8Validation)
	metrics, err := promParser.TextToMetricFamilies(rec.Body)
	require.NoError(t, err, "failed parsing metrics file")
	return metrics
}
