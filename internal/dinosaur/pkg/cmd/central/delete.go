package central

import (
	"fmt"

	"github.com/stackrox/rox/pkg/httputil"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/flags"
)

// NewDeleteCommand command for deleting centrals.
func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a central request",
		Long:  "Delete a central request.",
		Run: func(cmd *cobra.Command, args []string) {
			runDelete(fleetmanagerclient.AuthenticatedClientWithOCM(), cmd, args)
		},
	}

	cmd.Flags().String(FlagID, "", "Central ID")
	return cmd
}

func runDelete(client *fleetmanager.Client, cmd *cobra.Command, _ []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())

	const async = true
	resp, err := client.PublicAPI().DeleteCentralById(cmd.Context(), id, async)
	if err != nil {
		glog.Error(err)
		return
	}

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		glog.Errorf(apiErrorMsg, resp.Status)
		return
	}

	fmt.Printf("{status_code: %d}", resp.StatusCode)
}
