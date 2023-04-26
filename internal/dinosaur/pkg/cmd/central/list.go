package central

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// FlagPage ...
const (
	FlagPage = "page"
	FlagSize = "size"
)

// NewListCommand creates a new command for listing centrals.
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed central requests",
		Long:  "lists all managed central requests",
		Run: func(cmd *cobra.Command, args []string) {
			runList(fleetmanagerclient.AuthenticatedClientWithOCM(), cmd, args)
		},
	}
	cmd.Flags().String(FlagOwner, "test-user", "Username")
	cmd.Flags().String(FlagPage, "1", "Page index")
	cmd.Flags().String(FlagSize, "100", "Number of central requests per page")

	return cmd
}

func runList(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	centrals, _, err := client.PublicAPI().GetCentrals(cmd.Context(), &public.GetCentralsOpts{})
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
