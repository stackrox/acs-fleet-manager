// Package main provides a command line utility for running dataplane migrations.
package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	dp_migrator "github.com/stackrox/acs-fleet-manager/dp-migrator"
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

	rootCmd := &cobra.Command{
		Use:  "dataplane-migrator",
		Long: "Helper utility for running migrations on dataplane clusters.",
	}

	rootCmd.AddCommand(dp_migrator.Command())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
