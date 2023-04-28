package centrals

import "github.com/spf13/cobra"

func NewAdminCentralsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "centrals",
		Short:            "Perform admin central API calls.",
		Long:             "Perform admin central API calls.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(
		NewListCommand(),
	)

	return cmd
}
