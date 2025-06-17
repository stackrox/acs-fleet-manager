// Package cmd ...
package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"
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

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the gitops config.",
		Long:  "Validate the gitops config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("gitops file path expected as first argument")
			}

			gf := gitops.NewFileReader(args[0])
			cfg, err := gf.Read()
			if err != nil {
				return errors.Wrap(err, "failed to read gitops config")
			}

			errs := gitops.ValidateConfig(cfg)
			if len(errs) > 0 {
				return fmt.Errorf("validation failed: %v", errs.ToAggregate())
			}

			fmt.Println("validation successful")
			return nil
		},
	}
}
