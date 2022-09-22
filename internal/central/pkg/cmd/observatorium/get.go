package observatorium

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewRunGetStateCommand ...
func NewRunGetStateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-state",
		Short: "Fetch central state metric from Prometheus",
		Run: func(cmd *cobra.Command, args []string) {
			runGethResourceStateMetrics(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Central name")
	cmd.Flags().String(FlagNameSpace, "", "Central namespace")

	return cmd
}
func runGethResourceStateMetrics(env *environments.Env, cmd *cobra.Command, _args []string) {

	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	namespace := flags.MustGetDefinedString(FlagNameSpace, cmd.Flags())

	var observatoriumService services.ObservatoriumService
	env.MustResolveAll(&observatoriumService)

	centralState, err := observatoriumService.GetDinosaurState(name, namespace)
	if err != nil {
		glog.Error("An error occurred while attempting to fetch Observatorium data from Prometheus", err.Error())
		return
	}
	if len(centralState.State) > 0 {
		glog.Infof("central state is %s ", centralState.State)
	} else {
		glog.Infof("central state not found for parameters %s %s ", name, namespace)
	}

}
