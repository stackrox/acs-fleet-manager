package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/caarlos0/env/v6"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/runtime"
	"golang.org/x/sys/unix"
)

type Config struct {
	FleetManagerEndpoint string `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	ClusterID            string `env:"CLUSTER_ID"`
}

/**
- 1. setting up fleet-manager
- 2. calling API to get Centrals/Dinosaurs
- 3. Applying Dinosaurs into the Kubernetes API
- 4. Implement polling
- 5. Report status to fleet-manager
*/
func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	var config Config
	if err := env.Parse(&config); err != nil {
		glog.Fatalf("Unable to parse runtime configuration from environment: %v", err)
	}
	if config.ClusterID == "" {
		glog.Fatal("CLUSTER_ID unset in the environment")
	}
	if config.FleetManagerEndpoint == "" {
		glog.Fatal("FLEET_MANAGER_ENDPOINT unset in the environment")
	}

	glog.Infof("Starting application with configuration: %+v\n", config)

	runtime, err := runtime.NewRuntime(config.FleetManagerEndpoint, config.ClusterID, k8s.CreateClientOrDie())
	if err != nil {
		glog.Fatal(err)
	}

	go func() {
		err := runtime.Start()
		if err != nil {
			glog.Fatal(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	sig := <-sigs
	runtime.Stop()
	glog.Infof("Caught %s signal", sig)
	glog.Info("fleetshard application has been stopped")
}
