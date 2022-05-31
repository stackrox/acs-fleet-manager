package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/runtime"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
)

const (
	clusterID   = "1234567890abcdef1234567890abcdef"
	devEndpoint = "http://127.0.0.1:8000"
)

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

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	runtime := runtime.NewRuntime(devEndpoint, clusterID, k8s.CreateClientOrDie())
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
