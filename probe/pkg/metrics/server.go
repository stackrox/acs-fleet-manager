package metrics

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/rox/pkg/utils"
)

// NewMetricsServer returns the metrics server.
func NewMetricsServer(config config.Config) *http.Server {
	registry := initPrometheus(MetricsInstance())
	return newMetricsServer(config.MetricsAddress, registry)
}

// ListenAndServe listens for incoming requests and serves the metrics.
func ListenAndServe(server *http.Server) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		utils.Should(errors.Wrap(err, "failed to serve metrics"))
	}
}

// CloseMetricsServer closes the metrics server.
func CloseMetricsServer(server *http.Server) {
	if err := server.Close(); err != nil {
		utils.Should(errors.Wrap(err, "failed to close metrics server"))
	}
}

func initPrometheus(customMetrics *Metrics) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	// Register default metrics to use a dedicated registry instead of prometheus.DefaultRegistry.
	// This makes it easier to isolate metric state when unit testing this package.
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())
	customMetrics.Register(registry)
	return registry
}

func newMetricsServer(address string, registry *prometheus.Registry) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return &http.Server{Addr: address, Handler: mux}
}
