// Package main for email sender service
package main

import (
	"context"
	"errors"
	"flag"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"net/http"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/api"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/email"
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
		glog.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	dbCfg := cfg.DatabaseConfig.GetDbConfig()
	if err = dbCfg.ReadFiles(); err != nil {
		glog.Warningf("Failed to read DB configuration from files: %v", err)
		glog.Warning("Use DB configuration from plain environment variables")
	}

	ctx := context.Background()

	// initialize components
	dbConnection := db.NewDatabaseConnection(dbCfg)
	// TODO(ROX-23260): connect Rate Limiter to Email Sender
	_ = email.NewRateLimiterService(dbConnection)
	sesClient, err := email.NewSES(ctx, cfg.SesMaxBackoffDelay, cfg.SesMaxAttempts)
	if err != nil {
		glog.Errorf("Failed to initialise SES Client: %v", err)
		os.Exit(1)
	}

	emailSender := email.NewEmailSender(cfg.SenderAddress, sesClient)
	emailHandler := api.NewEmailHandler(emailSender)

	router, err := api.SetupRoutes(cfg.AuthConfig, emailHandler)
	if err != nil {
		glog.Errorf("Failed to set up router: %v", err)
		os.Exit(1)
	}

	server := http.Server{Addr: cfg.ServerAddress, Handler: router}

	go func() {
		glog.Info("Creating api server...")
		var err error
		if cfg.EnableHTTPS {
			err = server.ListenAndServeTLS(cfg.HTTPSCertFile, cfg.HTTPSKeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if !errors.Is(err, http.ErrServerClosed) {
			glog.Errorf("api server error: %v", err)
		}
	}()

	metricServer := metrics.NewMetricsServer(cfg.MetricsAddress)
	go func() {
		glog.Info("Creating metrics server...")
		if err := metricServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			glog.Errorf("metrics server error: %v", err)
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
	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server error: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Email sender application has been stopped")
}
