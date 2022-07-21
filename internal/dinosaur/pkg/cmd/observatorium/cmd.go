package observatorium

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// NewRunObservatoriumCommand ...
func NewRunObservatoriumCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observatorium",
		Short: "Perform observatorium actions directly",
		Long:  "Perform observatorium actions directly.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}

	// add sub-commands
	cmd.AddCommand(
		NewRunGetStateCommand(env),
		NewRunMetricsQueryRangeCommand(env),
		NewRunMetricsQueryCommand(env),
	)
	return cmd
}
