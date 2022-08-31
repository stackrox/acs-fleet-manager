package fleetshardmetrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer returns the metrics server
func NewMetricsServer(address string) *http.Server {
	registry := prometheus.NewRegistry()
	// Register default metrics to use a dedicated registry instead of prometheus.DefaultRegistry
	// this makes it easier to isolate metric state when unit testing this package
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())
	registerCustomMetrics(registry)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return &http.Server{Addr: address, Handler: mux}
}
