package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metricResponse map[string]*io_prometheus_client.MetricFamily

var cfg = &config.Config{
	MetricsAddress: ":7777",
}

func TestMetricsServerCorrectAddress(t *testing.T) {
	server := NewMetricsServer(cfg)
	defer server.Close()
	assert.Equal(t, ":7777", server.Addr)
}

func TestMetricsServerServesDefaultMetrics(t *testing.T) {
	registry := initPrometheus()
	metrics := serveMetrics(t, registry)
	assert.Contains(t, metrics, "go_memstats_alloc_bytes", "expected metrics to contain go default metrics but it did not")
}

func serveMetrics(t *testing.T, registry *prometheus.Registry) metricResponse {
	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err, "failed creating metrics requests")

	server := newMetricsServer(":7777", registry)
	defer server.Close()
	server.Handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "status code should be OK")

	promParser := expfmt.TextParser{}
	metrics, err := promParser.TextToMetricFamilies(rec.Body)
	require.NoError(t, err, "failed parsing metrics response")
	return metrics
}
