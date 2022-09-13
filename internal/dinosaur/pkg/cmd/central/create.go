package central

import (
	"encoding/json"

	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewCreateCommand creates a new command for creating centrals.
func NewCreateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new central request",
		Long:  "Create a new central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runCreate(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Central request name")
	cmd.Flags().String(FlagRegion, "us-east-1", "OCM region ID")
	cmd.Flags().String(FlagProvider, "aws", "OCM provider ID")
	cmd.Flags().String(FlagOwner, "test-user", "User name")
	cmd.Flags().String(FlagClusterID, "000", "Central request cluster ID")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Central request should be Multi AZ or not")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")

	return cmd
}

func runCreate(env *environments.Env, cmd *cobra.Command, _ []string) {
	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	multiAZ := flags.MustGetBool(FlagMultiAZ, cmd.Flags())
	clusterID := flags.MustGetDefinedString(FlagClusterID, cmd.Flags())
	orgID := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())

	var centralService services.DinosaurService
	env.MustResolveAll(&centralService)

	centralRequest := &dbapi.CentralRequest{
		Region:         region,
		ClusterID:      clusterID,
		CloudProvider:  provider,
		MultiAZ:        multiAZ,
		Name:           name,
		Owner:          owner,
		OrganisationID: orgID,
	}

	if err := centralService.RegisterDinosaurJob(centralRequest); err != nil {
		glog.Fatalf("Unable to create central request: %s", err.Error())
	}
	indentedCentralRequest, err := json.MarshalIndent(centralRequest, "", "    ")
	if err != nil {
		glog.Fatalf("Failed to format central request: %s", err.Error())
	}
	glog.V(10).Infof("%s", indentedCentralRequest)
}
