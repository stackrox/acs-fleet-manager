// Package metrics implements Prometheus metrics for email sender service
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "acs"
	prometheusSubsystem = "emailsender"
	clusterIDLabelName  = "cluster_id"
)

var (
	metrics *Metrics
	once    sync.Once
)

// Metrics holds the prometheus.Collector instances
type Metrics struct {
	emailsSent *prometheus.CounterVec
}

// Register registers the metrics with the given prometheus.Registerer.
func (m *Metrics) Register(r prometheus.Registerer) {
	r.MustRegister(m.emailsSent)
}

// IncEmailsSent increments the metric counter for started probe runs.
func (m *Metrics) IncEmailsSent(clusterID string) {
	m.emailsSent.With(prometheus.Labels{clusterIDLabelName: clusterID}).Inc()
}

// DefaultInstance returns the global Singleton instance for Metrics.
func DefaultInstance() *Metrics {
	once.Do(func() {
		metrics = newMetrics()
	})
	return metrics
}

func newMetrics() *Metrics {
	return &Metrics{
		emailsSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "email_sent_total",
			Help:      "The number of sent emails.",
		}, []string{clusterIDLabelName},
		),
	}
}
