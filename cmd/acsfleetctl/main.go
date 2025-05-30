// main package for acsfleetctl CLI
package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/admin"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/centrals"
	gitopsCmd "github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops/cmd"
)

func main() {
	rootCmd := &cobra.Command{
		Use:  "acsfleetctl",
		Long: "acsfleetctl is a CLI used to interact with the ACSCS fleet-manager API",
	}

	setupSubCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

}

func setupSubCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(centrals.NewCentralsCommand())
	rootCmd.AddCommand(admin.NewAdminCommand())
	rootCmd.AddCommand(gitopsCmd.NewGitOpsCommand())
}
