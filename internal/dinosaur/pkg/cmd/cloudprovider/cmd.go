// Package cloudprovider contains commands for interacting with cloud provider service directly instead of through the
// REST API exposed via the serve command.
package cloudprovider

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// NewCloudProviderCommand ...
func NewCloudProviderCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud_providers",
		Short: "Perform managed-services-api cloud providers actions directly",
		Long:  "Perform managed-services-api cloud providers actions directly.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}

	// add sub-commands
	cmd.AddCommand(
		NewProviderListCommand(env),
		NewRegionsListCommand(env),
	)

	return cmd
}
