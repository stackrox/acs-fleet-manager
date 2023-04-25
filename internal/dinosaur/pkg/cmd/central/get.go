package central

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewGetCommand gets a new command for getting centrals.
func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a central request",
		Long:  "Get a central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(fleetmanagerclient.AuthenticatedClientWithOCM(), cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Central ID")

	return cmd
}

func runGet(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())

	centralRequest, _, err := client.PublicAPI().GetCentralById(cmd.Context(), id)
	if err != nil {
		glog.Errorf(apiErrorMsg, "get", err)
		return
	}

	centralJSON, err := json.Marshal(centralRequest)
	if err != nil {
		glog.Error(err)
		return
	}
	fmt.Println(string(centralJSON))
}
