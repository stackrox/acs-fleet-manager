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
		glog.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// base router
	router := mux.NewRouter()

	// example handler
	router.HandleFunc("/test", func(rw http.ResponseWriter, req *http.Request) {
		glog.Info("called /test endpoint")
	})

	server := http.Server{Addr: cfg.ServerAddress, Handler: router}

	go func() {
		glog.Info("Creating api server...")
		var err error
		if cfg.EnableHTTPS {
			err = server.ListenAndServeTLS(cfg.HTTPSCertFile, cfg.HTTPSKeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != http.ErrServerClosed {
			glog.Fatalf("ListenAndServer error: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	notifySignals := []os.Signal{os.Interrupt, unix.SIGTERM}
	signal.Notify(sigs, notifySignals...)

	glog.Info("Application started. Will shut down gracefully on interrupt terminated OS signals")
	sig := <-sigs
	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("API Shutdown error: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Email sender application has been stopped")
}
