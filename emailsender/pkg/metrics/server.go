package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer returns the metrics server.
func NewMetricsServer(address string) *http.Server {
	registry := initPrometheus(NewInstance())
	return newMetricsServer(address, registry)
}

func initPrometheus(customMetrics *Metrics) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	// Register default metrics to use a dedicated registry instead of prometheus.DefaultRegistry.
	// This makes it easier to isolate metric state when unit testing this package.
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())
	customMetrics.Register(registry)
	return registry
}

func newMetricsServer(address string, registry *prometheus.Registry) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return &http.Server{Addr: address, Handler: mux}
}
