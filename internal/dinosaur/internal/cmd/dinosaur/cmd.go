// Package cluster contains commands for interacting with cluster logic of the service directly instead of through the
// REST API exposed via the serve command.
package dinosaur

import (
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewDinosaurCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dinosaur",
		Short: "Perform dinosaur CRUD actions directly",
		Long:  "Perform dinosaur CRUD actions directly.",
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
