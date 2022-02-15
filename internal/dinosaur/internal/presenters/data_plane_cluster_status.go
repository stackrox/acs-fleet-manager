package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
)

func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) (*dbapi.DataPlaneClusterStatus, error) {
	// TODO implement converter
	var res *dbapi.DataPlaneClusterStatus
	return res, nil
}

func PresentDataPlaneClusterConfig(config *dbapi.DataPlaneClusterConfig) private.DataplaneClusterAgentConfig {
	// TODO implement presenter
	var res private.DataplaneClusterAgentConfig
	return res
}
