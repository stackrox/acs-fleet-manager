package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/cmd"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true.")
	}

	var exitCode int
	defer func() { os.Exit(exitCode) }()

	cmd := cmd.Command()
	if err := cmd.Execute(); err != nil {
		exitCode = 1
	}
}
