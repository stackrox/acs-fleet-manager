package internal

import (
	"github.com/prometheus/client_golang/prometheus/testutil"
	"strings"
	"testing"
)

func TestGettingLeftClusterInstance(t *testing.T) {
	clusterConfig := &ClusterConfig{
		clusterConfigMap: map[string]ManualCluster{
			"cluster1": {Name: "Cluster1", ClusterID: "cluster1", CentralInstanceLimit: 10},
			"cluster2": {Name: "Cluster2", ClusterID: "cluster2", CentralInstanceLimit: 15},
			"cluster3": {Name: "Cluster3", ClusterID: "cluster3", CentralInstanceLimit: 20},
		},
	}
	clusterConfig.GettingLeftClusterInstance("cluster1", 5)
	GettingLeftClusterInstance("cluster2", 10)
	GettingLeftClusterInstance("cluster3", 20)

	expected := `
# HELP Central_instance_limit Spots left in Central Instance Limit
# TYPE Central_instance_limit gauge
Central_instance_limit{clusterId="cluster1"} 5
Central_instance_limit{clusterId="cluster2"} 5
Central_instance_limit{clusterId="cluster3"} 0
`
	if err := testutil.CollectAndCompare(CentralInstanceLimitMetric, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)

	}
}
