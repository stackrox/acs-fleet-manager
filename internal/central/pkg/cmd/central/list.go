package central

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

// FlagPage ...
const (
	FlagPage = "page"
	FlagSize = "size"
)

// NewListCommand creates a new command for listing centrals.
func NewListCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed central requests",
		Long:  "lists all managed central requests",
		Run: func(cmd *cobra.Command, args []string) {
			runList(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagOwner, "test-user", "Username")
	cmd.Flags().String(FlagPage, "1", "Page index")
	cmd.Flags().String(FlagSize, "100", "Number of central requests per page")

	return cmd
}

func runList(env *environments.Env, cmd *cobra.Command, _ []string) {
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	page := flags.MustGetString(FlagPage, cmd.Flags())
	size := flags.MustGetString(FlagSize, cmd.Flags())
	var centralService services.CentralService
	env.MustResolveAll(&centralService)

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

	centralList, paging, err := centralService.List(ctx, listArgs)
	if err != nil {
		glog.Fatalf("Unable to list central request: %s", err.Error())
	}

	// format output
	centralRequestList := public.CentralRequestList{
		Kind:  "CentralRequestList",
		Page:  int32(paging.Page),
		Size:  int32(paging.Size),
		Total: int32(paging.Total),
		Items: []public.CentralRequest{},
	}

	for _, centralRequest := range centralList {
		converted := presenters.PresentCentralRequest(centralRequest)
		centralRequestList.Items = append(centralRequestList.Items, converted)
	}

	output, marshalErr := json.MarshalIndent(centralRequestList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format central request list: %s", err.Error())
	}

	glog.V(10).Infof("%s", output)
}
