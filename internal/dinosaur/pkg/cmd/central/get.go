package central

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
	"github.com/stackrox/rox/pkg/httputil"
)

// NewGetCommand gets a new command for getting centrals.
func NewGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a central request",
		Long:  "Get a central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runGet(fleetmanagerclient.AuthenticatedClientWithOCM(), cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Central ID")

	return cmd
}

func runGet(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())

	centralRequest, resp, err := client.PublicAPI().GetCentralById(context.Background(), id)
	if err != nil {
		glog.Error(err)
		return
	}
	if httputil.Is2xxStatusCode(resp.StatusCode) {
		glog.Errorf(apiErrorMsg, resp.Status)
		return
	}

	centralJSON, err := json.Marshal(centralRequest)
	if err != nil {
		glog.Error(err)
		return
	}
	fmt.Println(string(centralJSON))
}
