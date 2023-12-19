package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

// ConvertDataPlaneClusterStatus ...
func ConvertDataPlaneClusterStatus(status private.DataPlaneClusterUpdateStatusRequest) dbapi.DataPlaneClusterStatus {
	var addonInstallations []dbapi.AddonInstallation
	for _, addon := range status.Addons {
		addonInstallations = append(addonInstallations, dbapi.AddonInstallation{
			ID:                  addon.Id,
			Version:             addon.Version,
			SourceImage:         addon.SourceImage,
			PackageImage:        addon.PackageImage,
			ParametersSHA256Sum: addon.ParametersSHA256Sum,
		})
	}

	return dbapi.DataPlaneClusterStatus{
		Addons: addonInstallations,
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
