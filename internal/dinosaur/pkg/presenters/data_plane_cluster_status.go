package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

// ConvertDataPlaneClusterStatus ...
func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) (*dbapi.DataPlaneClusterStatus, error) {
	var res dbapi.DataPlaneClusterStatus
	res.Conditions = make([]dbapi.DataPlaneClusterStatusCondition, len(status.Conditions))
	for i, cond := range status.Conditions {
		res.Conditions[i] = dbapi.DataPlaneClusterStatusCondition{
			Type:    cond.Type,
			Reason:  cond.Reason,
			Status:  cond.Status,
			Message: cond.Message,
		}
	}
	return &res, nil
}

// PresentDataPlaneClusterConfig ...
func PresentDataPlaneClusterConfig(config *dbapi.DataPlaneClusterConfig) private.DataplaneClusterAgentConfig {
	res := private.DataplaneClusterAgentConfig{
		Spec: private.DataplaneClusterAgentConfigSpec{
			Observability: private.DataplaneClusterAgentConfigSpecObservability{
				AccessToken: &config.Observability.AccessToken,
				Channel:     config.Observability.Channel,
				Repository:  config.Observability.Repository,
				Tag:         config.Observability.Tag,
			},
		},
	}

	return res
}
