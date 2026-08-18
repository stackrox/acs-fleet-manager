// Package admin contains all admin API related CLI commands.
package admin

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/admin/centrals"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/fleetmanagerclient"
	impl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
)

// NewAdminCommand creates a new admin command.
func NewAdminCommand() *cobra.Command {
	var (
		apiURL   string
		authType string
	)

	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Perform admin API calls.",
		Long: fmt.Sprintf(`Perform admin API calls against the fleet-manager admin API.

Auth credentials are resolved from environment variables based on --auth-type:
  %s:                 RHSSO_SERVICE_ACCOUNT_CLIENT_ID, RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET,
                         RHSSO_ENDPOINT (default https://sso.redhat.com), RHSSO_REALM (default redhat-external)
  %s:          STATIC_TOKEN
  %s: FLEET_MANAGER_TOKEN_FILE`,
			impl.RHSSOAuthName, impl.StaticTokenAuthName, impl.ServiceAccountTokenAuthName),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			auth, err := impl.NewAuth(ctx, authType, impl.OptionFromEnv())
			if err != nil {
				return fmt.Errorf("creating auth: %w", err)
			}
			client, err := impl.NewClient(apiURL, auth)
			if err != nil {
				return fmt.Errorf("creating fleet-manager client: %w", err)
			}
			cmd.SetContext(fleetmanagerclient.NewContext(ctx, client))
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://api.openshift.com",
		"Fleet Manager admin API base URL")
	cmd.PersistentFlags().StringVar(&authType, "auth-type", impl.StaticTokenAuthName,
		fmt.Sprintf("Auth type: %s, %s, or %s (credentials from env vars)",
			impl.RHSSOAuthName, impl.StaticTokenAuthName, impl.ServiceAccountTokenAuthName))

	cmd.AddCommand(
		centrals.NewAdminCentralsCommand(),
	)

	return cmd
}
