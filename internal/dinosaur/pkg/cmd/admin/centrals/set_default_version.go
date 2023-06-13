package centrals

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// NewAdminCentralsSetDefaultVersionCommand returns a new command to set the default version for centrals.
func NewAdminCentralsSetDefaultVersionCommand(client *fleetmanager.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default-version version",
		Short: "Set the default version for centrals",
		Long:  "Set the default version for centrals",
		Run: func(cmd *cobra.Command, args []string) {
			runSetDefaultVersion(client, cmd, args)
		},
		Args: cobra.ExactArgs(1),
	}
	return cmd
}

func runSetDefaultVersion(client *fleetmanager.Client, cmd *cobra.Command, args []string) {
	version := args[0]

	_, err := client.AdminAPI().SetCentralDefaultVersion(cmd.Context(), private.CentralDefaultVersion{})
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error calling fleet-manager API: %v\n", err)
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Central Default Version set to: %s\n", version)
}
