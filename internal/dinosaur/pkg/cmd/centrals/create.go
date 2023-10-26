package centrals

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewCreateCommand creates a new command for creating centrals.
func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new central request",
		Long:  "Create a new central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runCreate(fleetmanagerclient.AuthenticatedClientWithOCM(cmd.Context()), cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Central request name (required)")
	cmd.Flags().String(FlagRegion, "us-east-1", "OCM region ID (required)")
	cmd.Flags().String(FlagProvider, "aws", "OCM provider ID (required)")
	cmd.Flags().String(FlagOwner, "test-user", "User name")
	cmd.Flags().String(FlagClusterID, "000", "Central request cluster ID")
	cmd.Flags().Bool(FlagMultiAZ, true, "Whether Central request should be Multi AZ or not")
	cmd.Flags().String(FlagOrgID, "", "OCM org id")
	flags.MarkFlagRequired(FlagName, cmd)
	flags.MarkFlagRequired(FlagRegion, cmd)
	flags.MarkFlagRequired(FlagProvider, cmd)
	return cmd
}

func runCreate(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	region := flags.MustGetDefinedString(FlagRegion, cmd.Flags())
	provider := flags.MustGetDefinedString(FlagProvider, cmd.Flags())

	request := public.CentralRequestPayload{
		Region:        region,
		CloudProvider: provider,
		Name:          name,
		MultiAz:       true,
	}

	const async = true
	centralRequest, _, err := client.PublicAPI().CreateCentral(cmd.Context(), async, request)
	if err != nil {
		glog.Errorf(apiErrorMsg, "create", err)
		return
	}

	centralJSON, err := json.Marshal(centralRequest)
	if err != nil {
		glog.Errorf("Failed to marshal Central: %s", err)
		return
	}
	fmt.Println(string(centralJSON))
}
