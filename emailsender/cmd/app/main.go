// Package main ...
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/metrics"
)

func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default, required for glog.
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("unable to set logtostderr to true.")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.StartupTimeout)
	defer cancel()

	server := http.Server{Addr: cfg.ServerAddress}

	go func() {
		glog.Info("Creating api server...")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			glog.Fatalf("API ListenAndServe error: %v", err)
		}
	}()

	metricServer := metrics.NewMetricsServer(cfg)
	go func() {
		glog.Info("Creating metrics server...")
		if err := metricServer.ListenAndServe(); err != nil {
			glog.Errorf("serving metrics server error: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	notifySignals := []os.Signal{os.Interrupt, unix.SIGTERM}
	signal.Notify(sigs, notifySignals...)

	glog.Infof("Application started. Will shut down gracefully on %s.", notifySignals)
	sig := <-sigs
	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("API Shutdown error: %v", err)
	}
	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server error: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Email sender application has been stopped")
}
