// Package cmd ...
package cmd

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				glog.Errorf("gitops file path expected as first argument")
				os.Exit(1)
			}

			configs, err := operator.ReadConfigs(args[0])
			if err != nil {
				glog.Errorf("validation failed reading configs: %s", err)
				os.Exit(1)
			}

			fieldPath := &field.Path{}
			errors := operator.Validate(fieldPath, configs)
			if len(errors) > 0 {
				glog.Errorf("gitops validation failed")
				for _, err := range errors {
					glog.Error(err)
				}
				os.Exit(1)
			}

			fmt.Println("validation successful")
		},
	}
}
