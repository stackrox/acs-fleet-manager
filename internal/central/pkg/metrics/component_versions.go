package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
)

type versionsMetrics struct {
	dinosaurService         services.CentralService
	dinosaurOperatorVersion *prometheus.GaugeVec
	dinosaurVersion         *prometheus.GaugeVec
}

// RegisterVersionMetrics need to invoked when the server is started and dinosaurService is initialised
func RegisterVersionMetrics(dinosaurService services.CentralService) {
	m := newVersionMetrics(dinosaurService)
	// for tests. This function will be called multiple times when run integration tests because `prometheus` is singleton
	prometheus.Unregister(m)
	prometheus.MustRegister(m)
}

func newVersionMetrics(dinosaurService services.CentralService) *versionsMetrics {
	return &versionsMetrics{
		dinosaurService: dinosaurService,
		dinosaurOperatorVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dinosaur_operator_version",
			Help: `Reports the version of Dinosaur Operator in terms of seconds since the epoch.
The type 'actual' is the Dinosaur Operator version that is reported by fleetshard.
The type 'desired' is the desired Dinosaur Operator version that is set in the fleet-manager.
If the type is 'upgrade' it means the Dinosaur Operator is being upgraded.
`,
		}, []string{"cluster_id", "dinosaur_id", "type", "version"}),
		dinosaurVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dinosaur_version",
			Help: `Reports the version of Dinosaur in terms of seconds since the epoch.
The type 'actual' is the Dinosaur version that is reported by fleetshard.
The type 'desired' is the desired Dinosaur version that is set in the fleet-manager.
If the type is 'upgrade' it means the Dinosaur is being upgraded.
`,
		}, []string{"cluster_id", "dinosaur_id", "type", "version"}),
	}
}

// Describe ...
func (m *versionsMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.dinosaurOperatorVersion.WithLabelValues("", "", "", "").Desc()
}

// Collect ...
func (m *versionsMetrics) Collect(ch chan<- prometheus.Metric) {
	// list all the Dinosaur instances from dinosaurServices and generate metrics for each
	// the generated metrics will be put on the channel
	if versions, err := m.dinosaurService.ListComponentVersions(); err == nil {
		for _, v := range versions {
			// actual dinosaur operator version
			actualDinosaurOperatorMetric := m.dinosaurOperatorVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualCentralOperatorVersion)
			actualDinosaurOperatorMetric.Set(float64(time.Now().Unix()))
			ch <- actualDinosaurOperatorMetric
			// desired metric
			desiredDinosaurOperatorMetric := m.dinosaurOperatorVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredCentralOperatorVersion)
			desiredDinosaurOperatorMetric.Set(float64(time.Now().Unix()))
			ch <- desiredDinosaurOperatorMetric

			if v.CentralOperatorUpgrading {
				dinosaurOperatorUpgradingMetric := m.dinosaurOperatorVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredCentralOperatorVersion)
				dinosaurOperatorUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- dinosaurOperatorUpgradingMetric
			}

			// actual dinosaur version
			actualDinosaurMetric := m.dinosaurVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualCentralVersion)
			actualDinosaurMetric.Set(float64(time.Now().Unix()))
			ch <- actualDinosaurMetric
			// desired dinosaur version
			desiredDinosaurMetric := m.dinosaurVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredCentralVersion)
			desiredDinosaurMetric.Set(float64(time.Now().Unix()))
			ch <- desiredDinosaurMetric

			if v.CentralUpgrading {
				dinosaurUpgradingMetric := m.dinosaurVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredCentralVersion)
				dinosaurUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- dinosaurUpgradingMetric
			}
		}
	} else {
		logger.Logger.Errorf("failed to get component versions due to err: %v", err)
	}
}
