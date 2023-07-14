// Package fleetshardmetrics ...
package fleetshardmetrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
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
	centralDBClustersUsed       prometheus.Gauge
	centralDBClustersMax        prometheus.Gauge
	centralDBInstancesUsed      prometheus.Gauge
	centralDBInstancesMax       prometheus.Gauge
	centralDBSnapshotsUsed      prometheus.Gauge
	centralDBSnapshotsMax       prometheus.Gauge
	pauseReconcileInstances     *prometheus.GaugeVec
	operatorsHealthStatus       *prometheus.GaugeVec
}

// Register registers the metrics with the given prometheus.Registerer
func (m *Metrics) Register(r prometheus.Registerer) {
	r.MustRegister(m.fleetManagerRequestErrors)
	r.MustRegister(m.fleetManagerRequests)
	r.MustRegister(m.centralReconcilations)
	r.MustRegister(m.centralReconcilationErrors)
	r.MustRegister(m.activeCentralReconcilations)
	r.MustRegister(m.totalCentrals)
	r.MustRegister(m.centralDBClustersUsed)
	r.MustRegister(m.centralDBClustersMax)
	r.MustRegister(m.centralDBInstancesUsed)
	r.MustRegister(m.centralDBInstancesMax)
	r.MustRegister(m.centralDBSnapshotsUsed)
	r.MustRegister(m.centralDBSnapshotsMax)
	r.MustRegister(m.pauseReconcileInstances)
	r.MustRegister(m.operatorsHealthStatus)
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

// SetDatabaseAccountQuotas sets all the metrics related to database quotas
func (m *Metrics) SetDatabaseAccountQuotas(quotas cloudprovider.AccountQuotas) {
	if quota, found := quotas[cloudprovider.DBClusters]; found {
		m.centralDBClustersUsed.Set(float64(quota.Used))
		m.centralDBClustersMax.Set(float64(quota.Max))
	}
	if quota, found := quotas[cloudprovider.DBInstances]; found {
		m.centralDBInstancesUsed.Set(float64(quota.Used))
		m.centralDBInstancesMax.Set(float64(quota.Max))
	}
	if quota, found := quotas[cloudprovider.DBSnapshots]; found {
		m.centralDBSnapshotsUsed.Set(float64(quota.Used))
		m.centralDBSnapshotsMax.Set(float64(quota.Max))
	}
}

// SetPauseReconcileStatus sets the pause reconcile metric for a particular instance
func (m *Metrics) SetPauseReconcileStatus(instance string, pauseReconcileEnabled bool) {
	var pauseReconcileValue float64
	if pauseReconcileEnabled {
		pauseReconcileValue = 1.0
	}

	m.pauseReconcileInstances.With(prometheus.Labels{"instance": instance}).Set(pauseReconcileValue)
}

// SetOperatorHealthStatus sets the health status for specific operator image
func (m *Metrics) SetOperatorHealthStatus(image string, healthy bool) {
	//var healthyVal float64
	//if healthy {
	//	healthyVal = 1.0
	//}

	//m.operatorsHealthStatus.With(prometheus.Labels{"image": image}).Set(healthyVal)
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
		centralDBClustersUsed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "central_db_clusters_used",
			Help: "The current number of Central DB clusters in the cloud region of fleetshard-sync",
		}),
		centralDBClustersMax: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "central_db_clusters_max",
			Help: "The maximum number of Central DB clusters in the cloud region of fleetshard-sync",
		}),
		centralDBInstancesUsed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "central_db_instances_used",
			Help: "The current number of Central DB instances in the cloud region of fleetshard-sync",
		}),
		centralDBInstancesMax: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "central_db_instances_max",
			Help: "The maximum number of Central DB instances in the cloud region of fleetshard-sync",
		}),
		centralDBSnapshotsUsed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "central_db_snapshots_used",
			Help: "The current number of Central DB snapshots in the cloud region of fleetshard-sync",
		}),
		centralDBSnapshotsMax: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metricsPrefix + "central_db_snapshots_max",
			Help: "The maximum number of Central DB snapshots in the cloud region of fleetshard-sync",
		}),
		pauseReconcileInstances: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricsPrefix + "pause_reconcile_instances",
				Help: "The pause-reconcile annotation status of all the instances managed by fleetshard-sync",
			},
			[]string{"instance"},
		),
		operatorsHealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricsPrefix + "operator_health_status_images",
				Help: "The operator health status reports all operators images installed by fleetshard-sync",
			},
			[]string{"image"},
		),
	}
}
