package dinosaur

import (
	"encoding/json"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// NewCreateCommand creates a new command for creating dinosaurs.
func NewCreateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new dinosaur request",
		Long:  "Create a new dinosaur request.",
		Run: func(cmd *cobra.Command, args []string) {
			runCreate(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Dinosaur request name")
	cmd.Flags().String(FlagRegion, "us-east-1", "OCM region ID")
	cmd.Flags().String(FlagProvider, "aws", "OCM provider ID")
	cmd.Flags().String(FlagOwner, "test-user", "User name")
	cmd.Flags().String(FlagClusterID, "000", "Dinosaur  request cluster ID")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Dinosaur request should be Multi AZ or not")
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
	orgId := flags.MustGetDefinedString(FlagOrgID, cmd.Flags())

	var dinosaurService services.DinosaurService
	env.MustResolveAll(&dinosaurService)

	dinosaurRequest := &dbapi.DinosaurRequest{
		Region:         region,
		ClusterID:      clusterID,
		CloudProvider:  provider,
		MultiAZ:        multiAZ,
		Name:           name,
		Owner:          owner,
		OrganisationId: orgId,
	}

	if err := dinosaurService.RegisterDinosaurJob(dinosaurRequest); err != nil {
		glog.Fatalf("Unable to create dinosaur request: %s", err.Error())
	}
	indentedDinosaurRequest, err := json.MarshalIndent(dinosaurRequest, "", "    ")
	if err != nil {
		glog.Fatalf("Failed to format dinosaur request: %s", err.Error())
	}
	glog.V(10).Infof("%s", indentedDinosaurRequest)
}
