// Package main ...
package main

import (
	"flag"

	"github.com/stackrox/acs-fleet-manager/pkg/cmd/migrate"
	"github.com/stackrox/acs-fleet-manager/pkg/cmd/serve"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating that the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Infof("Unable to set logtostderr to true")
	}

	env, err := environments.New(environments.GetEnvironmentStrFromEnv(),
		dinosaur.ConfigProviders(),
	)
	if err != nil {
		glog.Fatalf("error initializing: %v", err)
	}
	defer env.Cleanup()

	rootCmd := &cobra.Command{
		Use:  "fleet-manager",
		Long: "fleet-manager is a service that exposes a Rest API to manage ACS Central instances.",
	}

	err = env.AddFlags(rootCmd.PersistentFlags())
	if err != nil {
		glog.Fatalf("Unable to add global flags: %s", err.Error())
	}

	rootCmd.AddCommand(migrate.NewMigrateCommand(env))
	rootCmd.AddCommand(serve.NewServeCommand(env))
	// Unsupported CLI commands. Eventually some of them can be removed.
	// rootCmd.AddCommand(central.NewCentralCommand(env))
	// rootCmd.AddCommand(cluster.NewClusterCommand(env))
	// rootCmd.AddCommand(cloudprovider.NewCloudProviderCommand(env))
	// rootCmd.AddCommand(observatorium.NewRunObservatoriumCommand(env))
	// rootCmd.AddCommand(errors.NewErrorsCommand(env))

	if err := rootCmd.Execute(); err != nil {
		glog.Fatalf("error running command: %v", err)
	}

	if err != nil {
		glog.Fatalf("Unable to initialize environment: %s", err.Error())
	}
}
