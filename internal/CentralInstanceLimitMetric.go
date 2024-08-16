package internal

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

// just gives me like the spots left in that cluster from the limit. Hopefully this is what we need for our metric
func (conf *ClusterConfig) GettingLeftClusterInstance(clusterId string, currentFillings int) {
	if cluster, exists := conf.clusterConfigMap[clusterId]; exists {
		limit := cluster.CentralInstanceLimit
		LeftClusterSpots := float64(limit - currentFillings)
		CentralInstanceLimitMetric.WithLabelValues(cluster.ClusterID).Set(float64(LeftClusterSpots))
	}
}

func main() {
	GettingLeftClusterInstance("cluster1", 10)
	GettingLeftClusterInstance("cluster2", 15)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Beg. to connect to port")
	http.ListenAndServe(":9091", nil)

}
