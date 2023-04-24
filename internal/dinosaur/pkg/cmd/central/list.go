package central

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/rox/pkg/httputil"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

// FlagPage ...
const (
	FlagPage = "page"
	FlagSize = "size"
)

// NewListCommand creates a new command for listing centrals.
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "lists all managed central requests",
		Long:  "lists all managed central requests",
		Run: func(cmd *cobra.Command, args []string) {
			runList(fleetmanagerclient.AuthenticatedClientWithOCM(), args)
		},
	}
	cmd.Flags().String(FlagOwner, "test-user", "Username")
	cmd.Flags().String(FlagPage, "1", "Page index")
	cmd.Flags().String(FlagSize, "100", "Number of central requests per page")

	return cmd
}

func runList(client *fleetmanager.Client, _ []string) {
	centrals, resp, err := client.PublicAPI().GetCentrals(context.Background(), &public.GetCentralsOpts{})
	if err != nil {
		glog.Error(err)
		return
	}
	if httputil.Is2xxStatusCode(resp.StatusCode) {
		glog.Errorf(apiErrorMsg, resp.Status)
		return
	}

	centralJSON, err := json.Marshal(centrals)
	if err != nil {
		glog.Error(err)
		return
	}

	fmt.Println(string(centralJSON))
}
