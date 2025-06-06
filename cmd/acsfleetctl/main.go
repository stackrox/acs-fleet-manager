// main package for acsfleetctl CLI
package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/admin"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/centrals"
	gitopsCmd "github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops/cmd"
)

func main() {
	defer glog.Flush()
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

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Infof("Unable to set logtostderr to true")
	}
}
