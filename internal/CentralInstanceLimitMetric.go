package internal

import "github.com/prometheus/client_golang/prometheus"

var (
	CentralInstanceLimitMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "Central-Instance_Limit",
		Help: "Spots left in Central Instance Limit",
	},
		[]string{"clusterId"},
	)
)

func init() {
	prometheus.MustRegister(CentralInstanceLimitMetric)
}

// DataplaneClusterConfig ...
type DataplaneClusterConfig struct {
	ClusterConfig *ClusterConfig `json:"clusters_config"`
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

// IsNumberOfDinosaurWithinClusterLimit ...
func (conf *ClusterConfig) IsNumberOfDinosaurWithinClusterLimit(clusterID string, count int) bool {
	if _, exist := conf.clusterConfigMap[clusterID]; exist {
		limit := conf.clusterConfigMap[clusterID].CentralInstanceLimit
		return limit == -1 || count <= limit
	}
	return true
}
