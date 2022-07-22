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
)

// NewProviderListCommand a new command for listing providers.
func NewProviderListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "lists all supported cloud providers",
		Long:  "lists all supported cloud providers",
		Run: func(cmd *cobra.Command, args []string) {
			env.MustInvoke(runProviderList)
		},
	}
	return cmd
}

func runProviderList(
	providerConfig *config.ProviderConfig,
	cloudProviderService services.CloudProvidersService,
) {

	cloudProviders, err := cloudProviderService.ListCloudProviders()
	if err != nil {
		glog.Fatalf("Unable to list cloud providers: %s", err.Error())
	}
	cloudProviderList := public.CloudProviderList{
		Kind:  "CloudProviderList",
		Total: int32(len(cloudProviders)),
		Size:  int32(len(cloudProviders)),
		Page:  int32(1),
	}

	supportedProviders := providerConfig.ProvidersConfig.SupportedProviders
	for i := range cloudProviders {
		cloudProvider := cloudProviders[i]
		_, cloudProvider.Enabled = supportedProviders.GetByName(cloudProvider.ID)
		converted := presenters.PresentCloudProvider(&cloudProvider)
		cloudProviderList.Items = append(cloudProviderList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(cloudProviderList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format cloud provider list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
