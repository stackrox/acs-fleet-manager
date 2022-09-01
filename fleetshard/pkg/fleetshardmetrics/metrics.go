package fleetshardmetrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const metricsPrefix = "acs_fleetshard_"

var (
	metrics *Metrics
	once    sync.Once
)

// Metrics holds the prometheus.Collector instances for fleetshard-sync's custom metrics
// and provides methods to interact with them.
type Metrics struct {
	fleetManagerRequests      prometheus.Counter
	fleetManagerRequestErrors prometheus.Counter
}

// Register registers the metrics with the given prometheus.Registerer
func (m *Metrics) Register(r prometheus.Registerer) {
	r.MustRegister(m.fleetManagerRequestErrors)
	r.MustRegister(m.fleetManagerRequests)
}

// IncrementFleetManagerRequests increments the metric counter for fleet-manager requests
func (m *Metrics) IncrementFleetManagerRequests() {
	m.fleetManagerRequests.Inc()
}

// IncrementFleetManagerRequestErrors increments the metric counter for fleet-manager request errors
func (m *Metrics) IncrementFleetManagerRequestErrors() {
	m.fleetManagerRequestErrors.Inc()
}

// MetricsInstance return the global Singleton instance for Metrics
func MetricsInstance() *Metrics {
	once.Do(initMetricsInstance)
	return metrics
}

func initMetricsInstance() {
	metrics = newMetrics()
}

func newMetrics() *Metrics {
	return &Metrics{
		fleetManagerRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Name: metricsPrefix + "total_fleet_manager_requests",
			Help: "The total number of requests send to fleet-manager",
		}),
		fleetManagerRequestErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: metricsPrefix + "total_fleet_manager_request_errors",
			Help: "The total number of request errors for requests send to fleet-manager",
		}),
	}
}
