package dinosaur

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewGetCommand gets a new command for getting dinosaurs.
func NewGetCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a RHACS request",
		Long:  "Get a RHACS request.",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Dinosaur id")

	return cmd
}

func runGet(env *environments.Env, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	var dinosaurService services.DinosaurService
	env.MustResolveAll(&dinosaurService)

	dinosaurRequest, err := dinosaurService.GetById(id)
	if err != nil {
		glog.Fatalf("Unable to get dinosaur request: %s", err.Error())
	}
	indentedDinosaurRequest, marshalErr := json.MarshalIndent(dinosaurRequest, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format dinosaur request: %s", marshalErr.Error())
	}
	glog.V(10).Infof("%s", indentedDinosaurRequest)
}
