package cmd

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
	"golang.org/x/sys/unix"
)

// Command builds the root CLI command.
func Command() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          os.Args[0],
		Long:         "Probe is a service that verifies the availability of ACS fleet manager.",
	}
	c.AddCommand(
		startCommand(),
		runCommand(),
	)
	return c
}

func startCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "start",
		Short:        "Start a continuous loop of probe runs.",
		Long:         "Start a continuous loop of probe runs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeProbe(true, cmd, args)
		},
	}
	return c
}

func runCommand() *cobra.Command {
	c := &cobra.Command{
		SilenceUsage: true,
		Use:          "run",
		Short:        "Run a single probe run.",
		Long:         "Run a single probe run.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeProbe(false, cmd, args)
		},
	}
	return c
}

func executeProbe(loop bool, cmd *cobra.Command, args []string) error {
	glog.Infof("Probe service has been started.")

	config, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "Failed to load configuration")
	}

	metricsServer := metrics.NewMetricsServer(config.MetricsAddress)
	defer cleanupMetricsServer(metricsServer)
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			glog.Errorf("Failed to serve metrics: %v", err)
		}
	}()

	fleetManagerClient, err := fleetmanager.New(config)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize probe")
	}
	runtime, err := runtime.New(config, fleetManagerClient)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize probe")
	}
	defer cleanupRuntime(runtime)

	// TODO: Add server for liveness and readiness probes here.

	isErr := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	go runtime.Run(isErr, sigs, loop)

	sig := <-sigs
	if sig != nil {
		glog.Infof("Caught %s signal.", sig)
		isErr <- true
	}
	glog.Info("Probe service has been stopped.")

	returnErr := <-isErr
	if returnErr {
		return errors.New("Probe service failed. Exiting with status code 1.")
	}
	return nil
}

func cleanupRuntime(runtime *runtime.Runtime) {
	err := runtime.Stop()
	if err != nil {
		glog.Errorf("Failed to close runtime: %v", err)
	}
}

func cleanupMetricsServer(metricsServer *http.Server) {
	if err := metricsServer.Close(); err != nil {
		glog.Errorf("Failed to close metrics server: %v", err)
	}
}
