// Package metrics implements Prometheus metrics to instrument probe runs.
package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "acs"
	prometheusSubsystem = "probe"
)

var (
	metrics *Metrics
	once    sync.Once
)

// Metrics holds the prometheus.Collector instances for the probe's custom metrics
// and provides methods to interact with them.
type Metrics struct {
	startedRuns            prometheus.Counter
	succeededRuns          prometheus.Counter
	failedRuns             prometheus.Counter
	lastStartedTimestamp   prometheus.Gauge
	lastSuccessTimestamp   prometheus.Gauge
	lastFailureTimestamp   prometheus.Gauge
	totalDurationHistogram prometheus.Histogram
}

// Register registers the metrics with the given prometheus.Registerer.
func (m *Metrics) Register(r prometheus.Registerer) {
	r.MustRegister(m.startedRuns)
	r.MustRegister(m.succeededRuns)
	r.MustRegister(m.failedRuns)
	r.MustRegister(m.totalDurationHistogram)
	r.MustRegister(m.lastStartedTimestamp)
	r.MustRegister(m.lastSuccessTimestamp)
	r.MustRegister(m.lastFailureTimestamp)
}

// IncStartedRuns increments the metric counter for started probe runs.
func (m *Metrics) IncStartedRuns() {
	m.startedRuns.Inc()
}

// IncSucceededRuns increments the metric counter for successful probe runs.
func (m *Metrics) IncSucceededRuns() {
	m.succeededRuns.Inc()
}

// IncFailedRuns increments the metric counter for failed probe runs.
func (m *Metrics) IncFailedRuns() {
	m.failedRuns.Inc()
}

// SetLastStartedTimestamp sets timestamp for the last started probe run.
func (m *Metrics) SetLastStartedTimestamp() {
	m.lastStartedTimestamp.SetToCurrentTime()
}

// SetLastSuccessTimestamp sets timestamp for the last successful probe run.
func (m *Metrics) SetLastSuccessTimestamp() {
	m.lastSuccessTimestamp.SetToCurrentTime()
}

// SetLastFailureTimestamp sets timestamp for the last failed probe run.
func (m *Metrics) SetLastFailureTimestamp() {
	m.lastFailureTimestamp.SetToCurrentTime()
}

// ObserveTotalDuration sets the total duration gauge for probe runs.
func (m *Metrics) ObserveTotalDuration(duration time.Duration) {
	m.totalDurationHistogram.Observe(duration.Seconds())
}

// MetricsInstance returns the global Singleton instance for Metrics.
func MetricsInstance() *Metrics {
	once.Do(func() {
		metrics = newMetrics()
	})
	return metrics
}

func newMetrics() *Metrics {
	return &Metrics{
		startedRuns: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "runs_started_total",
			Help:      "The number of started probe runs.",
		}),
		succeededRuns: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "runs_succeeded_total",
			Help:      "The number of successful probe runs.",
		}),
		failedRuns: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "runs_failed_total",
			Help:      "The number of failed probe runs.",
		}),
		lastStartedTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_started_timestamp",
			Help:      "The Unix timestamp of the last started probe run.",
		}),
		lastSuccessTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_success_timestamp",
			Help:      "The Unix timestamp of the last successful probe run.",
		}),
		lastFailureTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_failure_timestamp",
			Help:      "The Unix timestamp of the last failed probe run.",
		}),
		totalDurationHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "total_duration_seconds",
			Help:      "The total run duration in seconds.",
			Buckets:   prometheus.ExponentialBuckets(30, 2, 8),
		}),
	}
}
