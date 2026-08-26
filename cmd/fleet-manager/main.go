// Package main ...
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/admin"
	"github.com/stackrox/acs-fleet-manager/pkg/cmd/migrate"
	"github.com/stackrox/acs-fleet-manager/pkg/cmd/serve"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/central"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating that the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Infof("Unable to set logtostderr to true")
	}

	env, err := environments.New(environments.GetEnvironmentStrFromEnv(),
		central.ConfigProviders(),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error initializing: %v\n", err)
		os.Exit(1)
	}
	rootCmd := rootCommand(env)

	if err := env.AddFlags(rootCmd.PersistentFlags()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unable to add global flags: %v\n", err)
		os.Exit(1)
	}
	err = rootCmd.Execute()
	env.Cleanup()
	if err != nil {
		os.Exit(1)
	}
}

func rootCommand(env *environments.Env) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:  "fleet-manager",
		Long: "fleet-manager is a service that exposes a Rest API to manage ACS Central instances.",
	}

	rootCmd.AddCommand(admin.NewAdminCommand())
	rootCmd.AddCommand(migrate.NewMigrateCommand(env))
	rootCmd.AddCommand(serve.NewServeCommand(env))

	return rootCmd
}
