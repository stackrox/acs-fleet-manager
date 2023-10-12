// Package main ...
package main

import (
	"context"
	"flag"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/internal/cli"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("unable to set logtostderr to true.")
	}

	config, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}

	if metricsServer := metrics.NewMetricsServer(config.MetricsAddress, config.DataPlaneRegion); metricsServer != nil {
		defer metrics.CloseMetricsServer(metricsServer)
		go metrics.ListenAndServe(metricsServer)
	} else {
		glog.Fatal(errors.New("unable to start metrics server"))
	}

	c, err := cli.New(context.Background(), config)
	if err != nil {
		glog.Fatal(err)
	}
	cmd := c.Command()
	if err := cmd.Execute(); err != nil {
		glog.Fatal(err)
	}
}
