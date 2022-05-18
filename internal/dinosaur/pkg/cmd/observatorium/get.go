package observatorium

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewRunGetStateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-state",
		Short: "Fetch dinosaur state metric from Prometheus",
		Run: func(cmd *cobra.Command, args []string) {
			runGethResourceStateMetrics(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagName, "", "Dinosaur name")
	cmd.Flags().String(FlagNameSpace, "", "Dinosaur namepace")

	return cmd
}
func runGethResourceStateMetrics(env *environments.Env, cmd *cobra.Command, _args []string) {

	name := flags.MustGetDefinedString(FlagName, cmd.Flags())
	namespace := flags.MustGetDefinedString(FlagNameSpace, cmd.Flags())

	var observatoriumService services.ObservatoriumService
	env.MustResolveAll(&observatoriumService)

	dinosaurState, err := observatoriumService.GetDinosaurState(name, namespace)
	if err != nil {
		glog.Error("An error occurred while attempting to fetch Observatorium data from Prometheus", err.Error())
		return
	}
	if len(dinosaurState.State) > 0 {
		glog.Infof("dinosaur state is %s ", dinosaurState.State)
	} else {
		glog.Infof("dinosaur state not found for paramerters %s %s ", name, namespace)
	}

}
