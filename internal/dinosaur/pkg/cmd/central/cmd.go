// Package central contains commands for interacting with central logic of the service directly instead of through the
// REST API exposed via the serve command.
package central

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
)

// NewCentralCommand ...
func NewCentralCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "central",
		Short: "Perform central CRUD actions directly",
		Long:  "Perform central CRUD actions directly.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}

	// add sub-commands
	cmd.AddCommand(
		NewCreateCommand(env),
		NewGetCommand(env),
		NewDeleteCommand(env),
		NewListCommand(env),
	)

	return cmd
}
