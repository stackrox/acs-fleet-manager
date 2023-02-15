// Package main ...
package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/runtime"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/upgrader"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"os/signal"
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
	glog.Infof("AuthType: %s", config.AuthType)

	glog.Infof("ManagedDB.Enabled: %t", config.ManagedDB.Enabled)
	glog.Infof("ManagedDB.SecurityGroup: %s", config.ManagedDB.SecurityGroup)
	glog.Infof("ManagedDB.SubnetGroup: %s", config.ManagedDB.SubnetGroup)

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		panic(err.Error())
	}

	u := upgrader.NewCanaryUpgrader(k8s.CreateClientOrDie())
	go func() {
		//TODO: refactor to an indepedent component to re-use shared informers?
		err := u.DeploymentInformer(clientset)
		glog.Info(err)
	}()

	runtime, err := runtime.NewRuntime(config, u, k8s.CreateClientOrDie())
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
	runtime.Stop()
	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("fleetshard application has been stopped")
}
