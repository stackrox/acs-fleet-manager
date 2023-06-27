package converters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
)

// ConvertDinosaurRequest ...
func ConvertDinosaurRequest(request *dbapi.CentralRequest) []map[string]interface{} {
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
