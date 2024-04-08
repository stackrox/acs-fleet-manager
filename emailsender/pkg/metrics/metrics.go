// Package metrics implements Prometheus metrics for email sender service
package metrics

import (
	"sync"
)

var (
	metrics *Metrics
	once    sync.Once
)

// Metrics holds the prometheus.Collector instances
type Metrics struct{}

// MetricsInstance returns the global Singleton instance for Metrics.
func MetricsInstance() *Metrics {
	once.Do(func() {
		metrics = newMetrics()
	})
	return metrics
}

func newMetrics() *Metrics {
	return &Metrics{}
}
