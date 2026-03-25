// Package main ...
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/stackrox/acs-fleet-manager/internal/certmonitor"

	"github.com/golang/glog"
	cfg "github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/runtime"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/server/profiler"
	"golang.org/x/sys/unix"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	config, err := cfg.GetConfig()
	if err != nil {
		glog.Fatalf("Failed to load configuration: %v", err)
	}
	if err := flag.Set("v", config.LogVerbosity); err != nil {
		glog.Errorf("Unable to set glog verbosity: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.StartupTimeout)
	defer cancel()
	glog.Infof("Starting application, timeout=%s", config.StartupTimeout)
	glog.Infof("FleetManagerEndpoint: %s", config.FleetManagerEndpoint)
	glog.Infof("ClusterID: %s", config.ClusterID)
	glog.Infof("RuntimePollPeriod: %s", config.RuntimePollPeriod.String())
	glog.Infof("AuthType: %s", config.AuthType)
	glog.Infof("LogVerbosity: %s", config.LogVerbosity)
	glog.Infof("ManagedDB.Enabled: %t", config.ManagedDB.Enabled)
	glog.Infof("ManagedDB.SecurityGroup: %s", config.ManagedDB.SecurityGroup)
	glog.Infof("ManagedDB.SubnetGroup: %s", config.ManagedDB.SubnetGroup)
	glog.Infof("TenantDefaultArgoCdAppSourceTargetRevision: %s", config.TenantDefaultArgoCdAppSourceTargetRevision)
	if len(config.TenantImagePullSecret) > 0 {
		glog.Infof("Image pull secret configured, will be injected into tenant namespaces.")
	}
	glog.Info("Creating k8s client...")
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		glog.Fatalf("Failed to get k8s config: %v", err)
	}
	k8sClient := k8s.CreateClientWithConfigOrDie(restConfig)
	ctrl.SetLogger(logger.NewKubeAPILogger())
	glog.Info("Creating runtime...")
	runtime, err := runtime.NewRuntime(ctx, config, k8sClient)
	if err != nil {
		glog.Fatal(err)
	}
	runtimeCtx, stopRuntime := context.WithCancel(context.Background())
	go func() {
		err := runtime.Start(runtimeCtx)
		if err != nil {
			glog.Fatal(err)
		}
	}()

	glog.Info("Creating certMonitor")

	monitor := certmonitor.NewCertMonitor(restConfig)
	if err := monitor.Start(); err != nil {
		glog.Fatalf("Error starting certmonitor: %v", err)
	}

	glog.Info("Creating metrics server...")
	metricServer := fleetshardmetrics.NewMetricsServer(config.MetricsAddress)
	go func() {
		if err := metricServer.ListenAndServe(); err != nil {
			glog.Errorf("serving metrics server: %v", err)
		}
	}()

	pprofServer := profiler.SingletonPprofServer()
	pprofServer.Start()
	defer pprofServer.Stop()

	sigs := make(chan os.Signal, 1)
	notifySignals := []os.Signal{os.Interrupt, unix.SIGTERM}
	signal.Notify(sigs, notifySignals...)

	glog.Infof("Application started. Will shut down gracefully on %s.", notifySignals)
	sig := <-sigs
	stopRuntime()
	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server: %v", err)
	}

	if err := monitor.Stop(); err != nil {
		glog.Errorf("Error stoping certmonitor: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Fleetshard-sync application has been stopped")
}
