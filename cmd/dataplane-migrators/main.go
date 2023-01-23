// Package main provides a command line utility for running dataplane migrations.
package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/dataplanemigrators"
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

	rootCmd.AddCommand(dataplanemigrators.Commands()...)

	if err := rootCmd.Execute(); err != nil {
		glog.Fatalf("error running command: %v", err)
	}
}
