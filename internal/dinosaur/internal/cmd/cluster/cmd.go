// Package cluster contains commands for interacting with cluster logic of the service directly instead of through the
// REST API exposed via the serve command.
package cluster

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewClusterCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Perform managed-services-api cluster actions directly",
		Long:  "Perform managed-services-api cluster actions directly.",
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
		NewScaleCommand(env),
	)

	return cmd
}
