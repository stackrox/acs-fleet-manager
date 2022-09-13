// Package cluster contains commands for interacting with cluster logic of the service directly instead of through the
// REST API exposed via the serve command.
package cluster

import (
	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// NewClusterCommand ...
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
