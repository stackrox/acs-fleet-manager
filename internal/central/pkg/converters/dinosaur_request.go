package converters

import (
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
)

// ConvertCentralRequest converts a dbapi.CentralRequest to a map.
func ConvertCentralRequest(request *dbapi.CentralRequest) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":             request.ID,
			"region":         request.Region,
			"cloud_provider": request.CloudProvider,
			"multi_az":       request.MultiAZ,
			"name":           request.Name,
			"status":         request.Status,
			"owner":          request.Owner,
			"cluster_id":     request.ClusterID,
			"host":           request.Host,
			"created_at":     request.Meta.CreatedAt,
			"updated_at":     request.Meta.UpdatedAt,
			"deleted_at":     request.Meta.DeletedAt.Time,
		},
	}
}

// ConvertCentralRequestList converts a CentralList to the response type expected by mocket
func ConvertCentralRequestList(centralList dbapi.CentralList) []map[string]interface{} {
	var centralRequestList []map[string]interface{}

	for _, centralRequest := range centralList {
		data := ConvertCentralRequest(centralRequest)
		centralRequestList = append(centralRequestList, data...)
	}

	return centralRequestList
}
