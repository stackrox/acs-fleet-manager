package central

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewGetCommand gets a new command for getting centrals.
func NewGetCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a central request",
		Long:  "Get a central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Central ID")

	return cmd
}

func runGet(env *environments.Env, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	var centralService services.DinosaurService
	env.MustResolveAll(&centralService)

	centralRequest, err := centralService.GetByID(id)
	if err != nil {
		glog.Fatalf("Unable to get central request: %s", err.Error())
	}
	indentedCentralRequest, marshalErr := json.MarshalIndent(centralRequest, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format central request: %s", marshalErr.Error())
	}
	glog.V(10).Infof("%s", indentedCentralRequest)
}
