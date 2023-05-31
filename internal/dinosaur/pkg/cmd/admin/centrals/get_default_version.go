package centrals

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// NewAdminCentralsGetDefaultVersionCommand returns a new command to get the default version for centrals.
func NewAdminCentralsGetDefaultVersionCommand(client *fleetmanager.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-default-version",
		Short: "Get the default version for centrals",
		Long:  "Get the default version for centrals",
		Run: func(cmd *cobra.Command, args []string) {
			runGetDefaultVersion(client, cmd, args)
		},
	}
	return cmd
}

func runGetDefaultVersion(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	defaultVersion, _, err := client.AdminAPI().GetCentralDefaultVersion(cmd.Context())
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "error calling fleet-manager API: %v\n", err)
		return
	}

	printJSON(cmd.OutOrStdout(), &defaultVersion)
}
