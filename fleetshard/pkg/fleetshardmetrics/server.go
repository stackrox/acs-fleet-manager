package fleetshardmetrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer returns the metrics server
func NewMetricsServer(address string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	registerCustomMetrics(prometheus.DefaultRegisterer)
	server := &http.Server{Addr: address, Handler: mux}
	return server
}
