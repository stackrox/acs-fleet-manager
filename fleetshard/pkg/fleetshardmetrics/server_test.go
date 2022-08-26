package fleetshardmetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

func TestMetricsServerCorrectAddress(t *testing.T) {
	server := NewMetricsServer(":8081")
	assert.Equal(t, ":8081", server.Addr)
}

func TestMetricsServerServesDefaultMetrics(t *testing.T) {
	server := NewMetricsServer(":8081")

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	assert.NoError(t, err, "failed creating metrics requests")

	server.Handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

	promParser := expfmt.TextParser{}
	metrics, err := promParser.TextToMetricFamilies(rec.Body)
	assert.NoError(t, err, "failed parsing metrics file")

	_, hasKey := metrics["go_memstats_alloc_bytes"]
	assert.Truef(t, hasKey, "expected metrics to contain go default metrics but it did not: %v", metrics)
}

func TestMetricsServerServesCustomMetrics(t *testing.T) {
	server := NewMetricsServer(":8081")

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	assert.NoError(t, err, "failed creating metrics requests")

	server.Handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

	promParser := expfmt.TextParser{}
	metrics, err := promParser.TextToMetricFamilies(rec.Body)
	assert.NoError(t, err, "failed creating metrics requests")
	assert.NoError(t, err, "failed parsing metrics file")

	expectedKeys := []string{
		"total_k8s_requests",
		"total_k8s_request_errors",
		"total_fleet_manager_requests",
		"total_fleet_manager_request_errors",
	}
	for _, key := range expectedKeys {
		_, hasKey := metrics[key]
		assert.Truef(t, hasKey, "expected metrics to contain %s but it did not: %v", key, metrics)
	}
}
