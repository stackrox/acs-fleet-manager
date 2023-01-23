package main

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/dataplanemigrators"
)

func main() {
	rootCmd := &cobra.Command{
		Use:  "dataplane-migrator",
		Long: "Helper utility for running migrations on dataplane clusters.",
	}

	rootCmd.AddCommand(dataplanemigrators.Commands()...)

	if err := rootCmd.Execute(); err != nil {
		glog.Fatalf("error running command: %v", err)
	}
}
