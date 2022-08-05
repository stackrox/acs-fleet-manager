package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
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
	res.AvailableDinosaurOperatorVersions = make([]api.CentralOperatorVersion, len(status.CentralOperator))
	for i, op := range status.CentralOperator {
		res.AvailableDinosaurOperatorVersions[i] = api.CentralOperatorVersion{
			Version: op.Version,
			Ready:   op.Ready,
		}
		res.AvailableDinosaurOperatorVersions[i].CentralVersions = make([]api.CentralVersion, len(op.CentralVersions))
		for j, v := range op.CentralVersions {
			res.AvailableDinosaurOperatorVersions[i].CentralVersions[j] = api.CentralVersion{Version: v}
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
