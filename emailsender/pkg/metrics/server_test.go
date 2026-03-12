package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metricResponse map[string]*io_prometheus_client.MetricFamily

func TestMetricsServerCorrectAddress(t *testing.T) {
	server := NewMetricsServer(cfg.MetricsAddress)
	assert.Equal(t, ":9999", server.Addr)
}

func TestMetricsServerServesDefaultMetrics(t *testing.T) {
	registry := initPrometheus(DefaultInstance())
	metrics := serveMetrics(t, registry)
	assert.Contains(t, metrics, "go_memstats_alloc_bytes", "not found default metrics")
}

func serveMetrics(t *testing.T, registry *prometheus.Registry) metricResponse {
	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err, "failed creating metrics requests")

	server := newMetricsServer(":9999", registry)
	server.Handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

	promParser := expfmt.NewTextParser(model.UTF8Validation)
	metrics, err := promParser.TextToMetricFamilies(rec.Body)
	require.NoError(t, err, "failed parsing metrics response")
	return metrics
}
