package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

// ConvertDataPlaneClusterStatus ...
func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) dbapi.DataPlaneClusterStatus {
	return dbapi.DataPlaneClusterStatus{
		FleetshardAddonStatus: dbapi.FleetshardAddonStatus{
			Version:             status.FleetshardAddonStatus.Version,
			SourceImage:         status.FleetshardAddonStatus.SourceImage,
			PackageImage:        status.FleetshardAddonStatus.PackageImage,
			ParametersSHA256Sum: status.FleetshardAddonStatus.ParametersSHA256Sum,
		},
	}
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
