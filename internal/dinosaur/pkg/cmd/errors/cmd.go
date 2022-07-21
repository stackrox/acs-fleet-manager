// Package errors contains commands for inspecting the list of errors which can be returned by the service
package errors

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// NewErrorsCommand ...
func NewErrorsCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Inspect the errors which can be returned by the service",
		Long:  "Inspect the errors which can be returned by the service",
	}

	// add sub-commands
	cmd.AddCommand(NewListCommand(env))

	return cmd
}
