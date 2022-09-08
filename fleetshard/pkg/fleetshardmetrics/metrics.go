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
	fleetManagerRequests        prometheus.Counter
	fleetManagerRequestErrors   prometheus.Counter
	centralReconcilations       prometheus.Counter
	centralReconcilationErrors  prometheus.Counter
	activeCentralReconcilations prometheus.Gauge
	totalCentrals               prometheus.Gauge
}

// Register registers the metrics with the given prometheus.Registerer
func (m *Metrics) Register(r prometheus.Registerer) {
	r.MustRegister(m.fleetManagerRequestErrors)
	r.MustRegister(m.fleetManagerRequests)
	r.MustRegister(m.centralReconcilations)
	r.MustRegister(m.centralReconcilationErrors)
	r.MustRegister(m.activeCentralReconcilations)
	r.MustRegister(m.totalCentrals)
}

// IncFleetManagerRequests increments the metric counter for fleet-manager requests
func (m *Metrics) IncFleetManagerRequests() {
	m.fleetManagerRequests.Inc()
}

// IncFleetManagerRequestErrors increments the metric counter for fleet-manager request errors
func (m *Metrics) IncFleetManagerRequestErrors() {
	m.fleetManagerRequestErrors.Inc()
}

// IncCentralReconcilations increments the metric counter for central reconcilations errors
func (m *Metrics) IncCentralReconcilations() {
	m.centralReconcilations.Inc()
}

// IncCentralReconcilationErrors increments the metric counter for central reconcilation errors
func (m *Metrics) IncCentralReconcilationErrors() {
	m.centralReconcilationErrors.Inc()
}

// SetTotalCentrals sets the metric for total centrals to the given value
func (m *Metrics) SetTotalCentrals(v float64) {
	m.totalCentrals.Set(v)
}

// IncActiveCentralReconcilations increments the metric gauge for active central reconcilations
func (m *Metrics) IncActiveCentralReconcilations() {
	m.activeCentralReconcilations.Inc()
}

// DecActiveCentralReconcilations decrements the metric gauge for active central reconcilations
func (m *Metrics) DecActiveCentralReconcilations() {
	m.activeCentralReconcilations.Dec()
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
		centralReconcilations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: metricsPrefix + "total_central_reconcilations",
			Help: "The total number of central reconcilations started",
		}),
		centralReconcilationErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: metricsPrefix + "total_central_reconcilation_errors",
			Help: "The total number of failed central reconcilations",
		}),
		activeCentralReconcilations: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "active_central_reconcilations",
			Help: "The number of currently running central reconcilations",
		}),
		totalCentrals: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "total_centrals",
			Help: "The total number of centrals monitored by fleetshard-sync",
		}),
	}
}
