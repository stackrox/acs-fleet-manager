// Package main for email sender service
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/mux"

	"golang.org/x/sys/unix"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
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

	// base router
	router := mux.NewRouter()

	// example handler
	router.HandleFunc("/test", func(rw http.ResponseWriter, req *http.Request) {
		glog.Info("called /test endpoint")
	})

	server := http.Server{Addr: cfg.ServerAddress, Handler: router}

	go func() {
		glog.Info("Creating api server...")
		if cfg.EnableHTTPS {
			if err := server.ListenAndServeTLS(cfg.HTTPSCertFile, cfg.HTTPSKeyFile); err != http.ErrServerClosed {
				glog.Fatalf("HTTPS API ListenAndServe error: %v", err)
			}
		} else {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				glog.Fatalf("HTTP API ListenAndServe error: %v", err)
			}
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

	glog.Infof("Caught %s signal", sig)
	glog.Info("Email sender application has been stopped")
}
