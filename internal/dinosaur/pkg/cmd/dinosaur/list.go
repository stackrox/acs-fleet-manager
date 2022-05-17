package dinosaur

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

const (
	FlagPage = "page"
	FlagSize = "size"
)

// NewListCommand creates a new command for listing dinosaurs.
func NewListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed dinosaur requests",
		Long:  "lists all managed dinosaur requests",
		Run: func(cmd *cobra.Command, args []string) {
			runList(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagOwner, "test-user", "Username")
	cmd.Flags().String(FlagPage, "1", "Page index")
	cmd.Flags().String(FlagSize, "100", "Number of dinosaur requests per page")

	return cmd
}

func runList(env *environments.Env, cmd *cobra.Command, _ []string) {
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	page := flags.MustGetString(FlagPage, cmd.Flags())
	size := flags.MustGetString(FlagSize, cmd.Flags())
	var dinosaurService services.DinosaurService
	env.MustResolveAll(&dinosaurService)

	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"username": owner,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)

	// build list arguments
	url := url.URL{}
	query := url.Query()
	query.Add(FlagPage, page)
	query.Add(FlagSize, size)
	listArgs := coreServices.NewListArguments(query)

	dinosaurList, paging, err := dinosaurService.List(ctx, listArgs)
	if err != nil {
		glog.Fatalf("Unable to list dinosaur request: %s", err.Error())
	}

	// format output
	dinosaurRequestList := public.CentralRequestList{
		Kind:  "DinosaurRequestList",
		Page:  int32(paging.Page),
		Size:  int32(paging.Size),
		Total: int32(paging.Total),
		Items: []public.CentralRequest{},
	}

	for _, dinosaurRequest := range dinosaurList {
		converted := presenters.PresentDinosaurRequest(dinosaurRequest)
		dinosaurRequestList.Items = append(dinosaurRequestList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(dinosaurRequestList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format dinosaur request list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
