// Package metrics implements Prometheus metrics for email sender service
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "acs"
	prometheusSubsystem = "emailsender"
	tenantIDLabelName   = "tenant_id"
)

var (
	metrics *Metrics
	once    sync.Once
)

// Metrics holds the prometheus.Collector instances
type Metrics struct {
	sendEmail          *prometheus.CounterVec
	failedSendEmail    *prometheus.CounterVec
	throttledSendEmail *prometheus.CounterVec
}

// Register registers the metrics with the given prometheus.Registerer
func (m *Metrics) Register(r prometheus.Registerer) {
	r.MustRegister(m.sendEmail)
	r.MustRegister(m.failedSendEmail)
	r.MustRegister(m.throttledSendEmail)
}

// IncSendEmail increments the metric counter for send email attempts
func (m *Metrics) IncSendEmail(tenantID string) {
	m.sendEmail.With(prometheus.Labels{tenantIDLabelName: tenantID}).Inc()
}

// IncFailedSendEmail increments the metric counter for fail send email attempts
func (m *Metrics) IncFailedSendEmail(tenantID string) {
	m.failedSendEmail.With(prometheus.Labels{tenantIDLabelName: tenantID}).Inc()
}

// IncThrottledSendEmail increments the metric counter for throttled send email
func (m *Metrics) IncThrottledSendEmail(tenantID string) {
	m.throttledSendEmail.With(prometheus.Labels{tenantIDLabelName: tenantID}).Inc()
}

// DefaultInstance returns the global Singleton instance for Metrics
func DefaultInstance() *Metrics {
	once.Do(func() {
		metrics = newMetrics()
	})
	return metrics
}

func newMetrics() *Metrics {
	return &Metrics{
		sendEmail: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "send_email_total",
			Help:      "The number of send email attempts.",
		}, []string{tenantIDLabelName},
		),
		failedSendEmail: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "failed_send_email_total",
			Help:      "The number of failed send email attempts.",
		}, []string{tenantIDLabelName},
		),
		throttledSendEmail: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "throttled_send_email_total",
			Help:      "The number of throttled send email attempts.",
		}, []string{tenantIDLabelName},
		),
	}
}
