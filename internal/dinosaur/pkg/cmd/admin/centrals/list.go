package centrals

import (
	"encoding/json"
	"fmt"

	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// NewAdminCentralsListCommand creates a new command for listing centrals.
func NewAdminCentralsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed central requests",
		Long:  "lists all managed central requests",
		Run: func(cmd *cobra.Command, args []string) {
			runList(fleetmanagerclient.AuthenticatedClientWithRHOASToken(), cmd, args)
		},
	}
	return cmd
}

func runList(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	centrals, _, err := client.AdminAPI().GetCentrals(cmd.Context(), &admin.GetCentralsOpts{})
	if err != nil {
		glog.Errorf(apiErrorMsg, "list", err)
		return
	}

	centralJSON, err := json.Marshal(centrals)
	if err != nil {
		glog.Errorf("Failed to marshal CentralRequests: %s", err)
		return
	}

	fmt.Println(string(centralJSON))
}
