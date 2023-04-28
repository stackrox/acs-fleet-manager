// Package central contains commands for interacting with central logic of the service directly instead of through the
// REST API exposed via the serve command.
package central

import (
	"github.com/spf13/cobra"
)

const (
	ApiErrorMsg = "%s Central failed: To fix this ensure you are authenticated, fleet-manager endpoint is configured and reachable. Status Code: %s."
)

// NewCentralCommand ...
func NewCentralCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "central",
		Short:            "Perform central CRUD actions directly",
		Long:             "Perform central CRUD actions directly.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}

	// add sub-commands
	cmd.AddCommand(
		NewCreateCommand(),
		NewGetCommand(),
		NewDeleteCommand(),
		NewListCommand(),
	)

	return cmd
}
