package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
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
func PresentDataPlaneClusterConfig(_ *dbapi.DataPlaneClusterConfig) private.DataplaneClusterAgentConfig {
	return private.DataplaneClusterAgentConfig{}
}
