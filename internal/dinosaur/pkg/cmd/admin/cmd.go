package admin

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/admin/centrals"
)

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
