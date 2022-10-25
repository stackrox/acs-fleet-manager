package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/runtime"
	"golang.org/x/sys/unix"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	config, err := config.GetConfig()
	if err != nil {
		glog.Fatalf("Failed to load configuration: %v", err)
	}

	glog.Info("Starting application")
	glog.Infof("FleetManagerEndpoint: %s", config.FleetManagerEndpoint)
	glog.Infof("ClusterID: %s", config.ClusterID)
	glog.Infof("RuntimePollPeriod: %s", config.RuntimePollPeriod.String())

	runtime, err := runtime.NewRuntime(config, k8s.CreateClientOrDie())
	if err != nil {
		glog.Fatal(err)
	}

	go func() {
		err := runtime.Start()
		if err != nil {
			glog.Fatal(err)
		}
	}()

	metricServer := fleetshardmetrics.NewMetricsServer(config.MetricsAddress)
	go func() {
		if err := metricServer.ListenAndServe(); err != nil {
			glog.Errorf("serving metrics server: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	sig := <-sigs

	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFn()
	err = runtime.Stop(ctx)
	if err != nil {
		glog.Error("Error shutting down", err)
	}

	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("fleetshard application has been stopped")
}
