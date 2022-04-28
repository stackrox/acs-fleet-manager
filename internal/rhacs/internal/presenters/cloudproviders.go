package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

func PresentCloudProvider(cloudProvider *api.CloudProvider) public.CloudProvider {
	reference := PresentReference(cloudProvider.Id, cloudProvider)
	return public.CloudProvider{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Name:        cloudProvider.Name,
		DisplayName: cloudProvider.DisplayName,
		Enabled:     cloudProvider.Enabled,
	}
}
func PresentCloudRegion(cloudRegion *api.CloudRegion) public.CloudRegion {
	reference := PresentReference(cloudRegion.Id, cloudRegion)
	return public.CloudRegion{
		Id:                     reference.Id,
		Kind:                   reference.Kind,
		DisplayName:            cloudRegion.DisplayName,
		Enabled:                cloudRegion.Enabled,
		SupportedInstanceTypes: cloudRegion.SupportedInstanceTypes,
	}
}
