package converters

import (
	"encoding/json"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

// ConvertCluster from *api.Cluster to []map[string]interface{}
func ConvertCluster(cluster *api.Cluster) []map[string]interface{} {
	var p, c *[]byte
	if cluster.ProviderSpec != nil {
		bytes, _ := json.Marshal(cluster.ProviderSpec)
		p = &bytes
	}
	if cluster.ClusterSpec != nil {
		bytes, _ := json.Marshal(cluster.ClusterSpec)
		c = &bytes
	}
	return []map[string]interface{}{
		{
			"id":             cluster.Meta.ID,
			"region":         cluster.Region,
			"cloud_provider": cluster.CloudProvider,
			"multi_az":       cluster.MultiAZ,
			"status":         cluster.Status,
			"cluster_id":     cluster.ClusterID,
			"external_id":    cluster.ExternalID,
			"created_at":     cluster.Meta.CreatedAt,
			"updated_at":     cluster.Meta.UpdatedAt,
			"deleted_at":     cluster.Meta.DeletedAt.Time,
			"provider_type":  cluster.ProviderType.String(),
			"provider_spec":  p,
			"cluster_spec":   c,
		},
	}
}

// ConvertClusterList - converts []api.Cluster to the response type expected by mocket
func ConvertClusterList(clusterList []api.Cluster) []map[string]interface{} {
	var convertedClusterList []map[string]interface{}

	for _, cluster := range clusterList {
		data := ConvertCluster(&cluster)
		convertedClusterList = append(convertedClusterList, data...)
	}

	return convertedClusterList
}

func ConvertClusters(clusterList []*api.Cluster) []map[string]interface{} {
	var convertedClusterList []map[string]interface{}

	for _, cluster := range clusterList {
		data := ConvertCluster(cluster)
		convertedClusterList = append(convertedClusterList, data...)
	}

	return convertedClusterList
}
