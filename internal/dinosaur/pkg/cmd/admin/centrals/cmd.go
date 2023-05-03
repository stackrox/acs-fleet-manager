// Package centrals contains the admin central CLI interface.
package centrals

import "github.com/spf13/cobra"

const (
	apiErrorMsg = "%s Admin Central failed: To fix this ensure you are authenticated, fleet-manager endpoint is configured and reachable. Status Code: %s."
)

// NewAdminCentralsCommand creates a new admin central command.
func NewAdminCentralsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "centrals",
		Short:            "Perform admin central API calls.",
		Long:             "Perform admin central API calls.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(
		NewListCommand(),
		// NewUpdateCommand(),
	)

	return cmd
}
