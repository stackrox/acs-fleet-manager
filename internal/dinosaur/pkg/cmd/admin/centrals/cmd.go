// Package centrals contains the admin central CLI interface.
package centrals

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/fleetmanagerclient"
)

const (
	apiErrorMsg = "%s admin Central failed: To fix this ensure you are authenticated, fleet-manager endpoint is configured and reachable. Status Code: %s."
)

// NewAdminCentralsCommand creates a new admin central command.
func NewAdminCentralsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "centrals",
		Aliases:          []string{"central"},
		Short:            "Perform admin central API calls.",
		Long:             "Perform admin central API calls.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	cmd.AddCommand(
		NewAdminCentralsListCommand(),
		NewAdminCentralsGetDefaultVersionCommand(fleetmanagerclient.AuthenticatedClientWithRHOASToken()),
	)

	return cmd
}

func printJSON(out io.Writer, data interface{}) {
	output, err := json.Marshal(data)
	if err != nil {
		glog.Errorf("Failed to marshal %T: %s", &data, err)
		return
	}

	fmt.Fprint(out, string(output))
}
