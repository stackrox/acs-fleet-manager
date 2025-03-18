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
	metrics         *Metrics
	once            sync.Once
	regionLabelName = "region"
)

// Metrics holds the prometheus.Collector instances for the probe's custom metrics
// and provides methods to interact with them.
type Metrics struct {
	startedRuns            *prometheus.CounterVec
	succeededRuns          *prometheus.CounterVec
	failedRuns             *prometheus.CounterVec
	lastStartedTimestamp   *prometheus.GaugeVec
	lastSuccessTimestamp   *prometheus.GaugeVec
	lastFailureTimestamp   *prometheus.GaugeVec
	totalDurationHistogram *prometheus.HistogramVec
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
func (m *Metrics) IncStartedRuns(region string) {
	m.startedRuns.With(prometheus.Labels{regionLabelName: region}).Inc()
}

// IncSucceededRuns increments the metric counter for successful probe runs.
func (m *Metrics) IncSucceededRuns(region string) {
	m.succeededRuns.With(prometheus.Labels{regionLabelName: region}).Inc()
}

// IncFailedRuns increments the metric counter for failed probe runs.
func (m *Metrics) IncFailedRuns(region string) {
	m.failedRuns.With(prometheus.Labels{regionLabelName: region}).Inc()
}

// SetLastStartedTimestamp sets timestamp for the last started probe run.
func (m *Metrics) SetLastStartedTimestamp(region string) {
	m.lastStartedTimestamp.With(prometheus.Labels{regionLabelName: region}).SetToCurrentTime()
}

// SetLastSuccessTimestamp sets timestamp for the last successful probe run.
func (m *Metrics) SetLastSuccessTimestamp(region string) {
	m.lastSuccessTimestamp.With(prometheus.Labels{regionLabelName: region}).SetToCurrentTime()
}

// SetLastFailureTimestamp sets timestamp for the last failed probe run.
func (m *Metrics) SetLastFailureTimestamp(region string) {
	m.lastFailureTimestamp.With(prometheus.Labels{regionLabelName: region}).SetToCurrentTime()
}

// ObserveTotalDuration sets the total duration gauge for probe runs.
func (m *Metrics) ObserveTotalDuration(duration time.Duration, region string) {
	m.totalDurationHistogram.With(prometheus.Labels{regionLabelName: region}).Observe(duration.Seconds())
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
		startedRuns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "runs_started_total",
			Help:      "The number of started probe runs.",
		}, []string{regionLabelName},
		),
		succeededRuns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "runs_succeeded_total",
			Help:      "The number of successful probe runs.",
		}, []string{regionLabelName},
		),
		failedRuns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "runs_failed_total",
			Help:      "The number of failed probe runs.",
		}, []string{regionLabelName},
		),
		lastStartedTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_started_timestamp",
			Help:      "The Unix timestamp of the last started probe run.",
		}, []string{regionLabelName},
		),
		lastSuccessTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_success_timestamp",
			Help:      "The Unix timestamp of the last successful probe run.",
		}, []string{regionLabelName},
		),
		lastFailureTimestamp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "last_failure_timestamp",
			Help:      "The Unix timestamp of the last failed probe run.",
		}, []string{regionLabelName},
		),
		totalDurationHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "total_duration_seconds",
			Help:      "The total run duration in seconds.",
			Buckets:   prometheus.ExponentialBuckets(30, 2, 8),
		}, []string{regionLabelName},
		),
	}
}
