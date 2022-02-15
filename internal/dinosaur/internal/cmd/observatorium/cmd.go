package observatorium

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

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
