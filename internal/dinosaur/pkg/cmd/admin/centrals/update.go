package centrals

import (
	"encoding/json"
	"fmt"

	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/central"
	fleetManagerFlags "github.com/stackrox/acs-fleet-manager/pkg/flags"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

const (
	CentralOperatorVersion = "central-operator-version" // TODO(sbaumer): move operator version to a separate command.
	ForceReconcile         = "force-reconcile"
)

// NewUpdateCommand creates a new command for updating centrals.
func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a Central request.",
		Long:  "Update a Central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runUpdate(fleetmanagerclient.AuthenticatedClientWithStaticToken(), cmd, args)
		},
	}

	cmd.Flags().String(central.FlagID, "", "Central ID")
	cmd.Flags().String(CentralOperatorVersion, "", "Central Operator Version")

	return cmd
}

func runUpdate(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	//TODO: Implement update command.
	glog.Fatal("Not implemented yet.")

	id := fleetManagerFlags.MustGetDefinedString(central.FlagID, cmd.Flags())
	centralOperatorVersion, _ := cmd.Flags().GetString(CentralOperatorVersion)
	forceReconcile, _ := cmd.Flags().GetString(ForceReconcile)

	request := admin.CentralUpdateRequest{
		CentralOperatorVersion: centralOperatorVersion,
		ForceReconcile:         forceReconcile,
	}

	centrals, _, err := client.AdminAPI().UpdateCentralById(cmd.Context(), id, request)
	if err != nil {
		glog.Errorf(ApiErrorMsg, "update", err)
		return
	}

	centralJSON, err := json.Marshal(centrals)
	if err != nil {
		glog.Errorf("Failed to marshal admin.Central list: %s", err)
		return
	}

	fmt.Println(string(centralJSON))
}
