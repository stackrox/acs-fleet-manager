package internal

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	apiErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"log"
	"net/http"
)

var (
	CentralInstanceLimitMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "Central_instance_limit",
		Help: "Spots left in Central Instance Limit",
	},
		[]string{"clusterId"},
	)
)

func init() {
	prometheus.MustRegister(CentralInstanceLimitMetric)
}

// ManualCluster manual cluster configuration
type ManualCluster struct {
	Name                 string `yaml:"name"`
	ClusterID            string `yaml:"cluster_id"`
	CentralInstanceLimit int    `yaml:"central_instance_limit"`
}

// ClusterConfig ...
type ClusterConfig struct {
	clusterConfigMap map[string]ManualCluster
}

type ClustermanagerOptions struct {
	DataplaneClusterConfig *config.DataplaneClusterConfig
	ClusterService         services.ClusterService
}

type ClusterManager struct {
	ClustermanagerOptions
}
type ResDinosaurInstanceCount struct {
	Clusterid string
	Count     int
}
type ClusterService interface {
	FindDinosaurInstanceCount(clusterIDs []string) ([]ResDinosaurInstanceCount, *apiErrors.ServiceError)
}

// just gives me like the spots left in that cluster from the limit. Hopefully this is what we need for our metric
func GettingLeftClusterInstance(clusterManager *ClusterManager) {
	for _, cluster := range clusterManager.DataplaneClusterConfig.ClusterConfig.GetManualClusters() {
		count, err := clusterManager.getClusterInstanceCount(clusterManager.ClusterID)
		if err != nil {
			log.Printf("Error getting count for cluster %s: %v", cluster.clusterID, err)
			continue
		}
		limit := cluster.CentralInstanceLimit
		LeftClusterSpots := float64(limit - count)
		CentralInstanceLimitMetric.WithLabelValues(cluster.ClusterID).Set(float64(LeftClusterSpots))
	}
}

func (clusterManager *ClusterManager) getClusterInstanceCount(clusterID string) (int, error) {
	counters, err := clusterManager.ClusterService.FindDinosaurInstanceCount([]string{clusterID})
	if err != nil {
		return 0, err
	}
	if len(counters) > 0 {
		return counters[0].Count, nil
	}
	return 0, nil
}

func main() {
	GettingLeftClusterInstance("cluster1", 10)
	GettingLeftClusterInstance("cluster2", 15)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Beg. to connect to port")
	http.ListenAndServe(":9091", nil)

}
