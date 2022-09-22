package cloudprovider

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewRegionsListCommand creates a new command for listing regions.
func NewRegionsListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regions",
		Short: "lists all supported cloud providers",
		Long:  "lists all supported cloud providers",
		Run: func(cmd *cobra.Command, args []string) {
			runRegionsList(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "aws", "Cloud provider id")
	cmd.Flags().String(FlagInstanceType, "", "Central instance type to filter the regions by")
	return cmd
}

func runRegionsList(env *environments.Env, cmd *cobra.Command, _ []string) {

	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	instanceTypeFilter := flags.MustGetString(FlagInstanceType, cmd.Flags())

	var providerConfig *config.ProviderConfig
	var cloudProviderService services.CloudProvidersService
	env.MustResolveAll(&providerConfig, &cloudProviderService)

	cloudRegions, err := cloudProviderService.ListCloudProviderRegions(id)
	if err != nil {
		glog.Fatalf("Unable to list cloud provider regions: %s", err.Error())
	}

	regionList := public.CloudRegionList{
		Kind:  "CloudRegionList",
		Total: int32(len(cloudRegions)),
		Size:  int32(len(cloudRegions)),
		Page:  int32(1),
	}

	supportedProviders := providerConfig.ProvidersConfig.SupportedProviders
	provider, _ := supportedProviders.GetByName(id)
	for i := range cloudRegions {
		cloudRegion := cloudRegions[i]
		region, _ := provider.Regions.GetByName(cloudRegion.ID)

		// if instance_type was specified, only set enabled to true for regions that supports the specified instance type. Otherwise,
		// set enable to true for all region that supports any instance types
		if instanceTypeFilter != "" {
			cloudRegion.Enabled = region.IsInstanceTypeSupported(config.InstanceType(instanceTypeFilter))
		} else {
			cloudRegion.Enabled = len(region.SupportedInstanceTypes) > 0
		}

		converted := presenters.PresentCloudRegion(&cloudRegion)
		regionList.Items = append(regionList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(regionList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format  cloud provider region list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)

}
