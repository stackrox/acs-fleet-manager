// Package admin contains all admin API related CLI commands.
package admin

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/admin/centrals"
)

// NewAdminCommand creates a new admin command.
func NewAdminCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "admin",
		Short:            "Perform admin API calls.",
		Long:             "Perform admin API calls. Use the STATIC_TOKEN to authenticate against its API.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(
		centrals.NewAdminCentralsCommand(),
	)

	return cmd
}
