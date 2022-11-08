package metrics

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer returns the metrics server.
func NewMetricsServer(address string) (*http.Server, func()) {
	metricsServer := newMetricsServer(address, MetricsInstance())
	closeFunc := func() {
		if err := metricsServer.Close(); err != nil {
			glog.Errorf("failed to close metrics server: %v", err)
		}
	}
	return metricsServer, closeFunc
}

func newMetricsServer(address string, customMetrics *Metrics) *http.Server {
	registry := prometheus.NewRegistry()
	// Register default metrics to use a dedicated registry instead of prometheus.DefaultRegistry.
	// This makes it easier to isolate metric state when unit testing this package.
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())
	customMetrics.Register(registry)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return &http.Server{Addr: address, Handler: mux}
}
