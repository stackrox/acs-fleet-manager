package internal

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"log"
	"net/http"
)

var (
	CentralInstanceLimitMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "Central_instance_limit",
		Help: "Spots left in Central Instance Limit",
	},
		[]string{"clusterId"})
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

func (conf *ClusterConfig) GetManualClusters() []ManualCluster {
	clusters := []ManualCluster{}
	for _, cluster := range conf.clusterConfigMap {
		clusters = append(clusters, cluster)
	}
	return clusters
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
	ClusterID string
	Count     int
}
type ClusterService interface {
	FindDinosaurInstanceCount(clusterIDs []string) ([]ResDinosaurInstanceCount, error)
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

// just gives me like the spots left in that cluster from the limit. Hopefully this is what we need for our metric
func GettingLeftClusterInstance(clusterManager *ClusterManager) {
	for _, cluster := range clusterManager.DataplaneClusterConfig.ClusterConfig.GetManualClusters() {
		count, err := clusterManager.getClusterInstanceCount(cluster.ClusterID)
		if err != nil {
			log.Printf("Error getting count for cluster %s: %v", cluster.ClusterID, err)
			continue
		}
		limit := cluster.CentralInstanceLimit
		LeftClusterSpots := float64(limit - count)
		CentralInstanceLimitMetric.WithLabelValues(cluster.ClusterID).Set(float64(LeftClusterSpots))
	}
}

type MockClusters struct{}

func (m *MockClusters) FindDinosaurInstanceCount(clusterIDs []string) ([]ResDinosaurInstanceCount, error) {
	return []ResDinosaurInstanceCount{
		{ClusterID: "cluster1", Count: 5},
		{ClusterID: "cluster2", Count: 7},
		{ClusterID: "cluster3", Count: 5},
	}, nil
}



func main() {
	clusterConfigMap := map[string]ManualCluster{
		"cluster1": {Name: "Cluster1", ClusterID: "cluster1", CentralInstanceLimit: 10},
		"cluster2": {Name: "Cluster2", ClusterID: "cluster2", CentralInstanceLimit: 15},
		"cluster3": {Name: "Cluster3", ClusterID: "cluster3", CentralInstanceLimit: 20},
	}

	clusterManager := &CentralInstanceMetric.ClusterManager{
		Config:
	}

	GettingLeftClusterInstance(clusterManager)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Beg. to connect to port")
	http.ListenAndServe(":9091", nil)

}