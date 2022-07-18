package dinosaur

import (
	"context"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewDeleteCommand command for deleting dinosaurs.
func NewDeleteCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a dinosaur request",
		Long:  "Delete a dinosaur request.",
		Run: func(cmd *cobra.Command, args []string) {
			runDelete(env, cmd, args)
		},
	}

	cmd.Flags().String(FlagID, "", "Dinosaur id")
	cmd.Flags().String(FlagOwner, "test-user", "Username")
	return cmd
}

func runDelete(env *environments.Env, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	var dinosaurService services.DinosaurService
	env.MustResolveAll(&dinosaurService)

	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"username": owner,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)

	if err := dinosaurService.RegisterDinosaurDeprovisionJob(ctx, id); err != nil {
		glog.Fatalf("Unable to register the deprovisioning request: %s", err.Error())
	} else {
		glog.V(10).Infof("Deprovisioning request accepted for dinosaur cluster with id %s", id)
	}
}
