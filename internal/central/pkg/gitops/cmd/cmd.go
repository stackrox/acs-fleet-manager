// Package cmd ...
package cmd

import (
	"github.com/spf13/cobra"
)

// NewGitOpsCommand creates a new gitops command.
func NewGitOpsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "gitops",
		Short:            "Perform actions like validation on the gitops config.",
		Long:             "Perform actions like validation on the gitops config.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(
		newValidateCommand(),
	)

	return cmd
}
