package presenters

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) (*dbapi.DataPlaneClusterStatus, error) {
	var res dbapi.DataPlaneClusterStatus
	res.Conditions = make([]dbapi.DataPlaneClusterStatusCondition, len(status.Conditions))
	for i, cond := range status.Conditions {
		res.Conditions[i] = dbapi.DataPlaneClusterStatusCondition{
			Type: cond.Type,
			Reason: cond.Reason,
			Status: cond.Status,
			Message: cond.Message,
		}
	}
	res.AvailableDinosaurOperatorVersions = make([]api.DinosaurOperatorVersion, len(status.DinosaurOperator))
	for i, op := range status.DinosaurOperator {
		res.AvailableDinosaurOperatorVersions[i] = api.DinosaurOperatorVersion{
			Version: op.Version,
			Ready: op.Ready,
		}
		res.AvailableDinosaurOperatorVersions[i].DinosaurVersions = make([]api.DinosaurVersion, len(op.DinosaurVersions))
		for j, v := range op.DinosaurVersions {
			res.AvailableDinosaurOperatorVersions[i].DinosaurVersions[j] = api.DinosaurVersion{Version: v}
		}
	}
	return &res, nil
}

func PresentDataPlaneClusterConfig(config *dbapi.DataPlaneClusterConfig) private.DataplaneClusterAgentConfig {
	// TODO implement presenter
	var res private.DataplaneClusterAgentConfig
	return res
}
