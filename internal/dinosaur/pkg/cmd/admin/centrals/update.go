package centrals

import (
	"encoding/json"
	"fmt"
	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/central"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// NewUpdateCommand creates a new command for updating centrals.
func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Update a Central request.",
		Long:  "Update a Central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runUpdate(fleetmanagerclient.AuthenticatedClientWithStaticToken(), cmd, args)
		},
	}

	cmd.Flags().String(central.FlagID, "", "Central ID")

	return cmd
}

func runUpdate(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(central.FlagID, cmd.Flags())

	centrals, _, err := client.AdminAPI().UpdateCentralById(cmd.Context(), id, admin.CentralUpdateRequest{})
	if err != nil {
		glog.Errorf(ApiErrorMsg, "list", err)
		return
	}

	centralJSON, err := json.Marshal(centrals)
	if err != nil {
		glog.Errorf("Failed to marshal CentralRequests: %s", err)
		return
	}

	fmt.Println(string(centralJSON))
}
