// Package main ...
package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/probe/internal/cli"
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

	c, err := cli.New()
	if err != nil {
		glog.Fatal(err)
	}
	cmd := c.Command()
	if err := cmd.Execute(); err != nil {
		glog.Fatal(err)
	}
}
