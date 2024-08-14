// Package main for email sender service
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/workers"

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
	shutdownCtx, cancelShutdownCtx := context.WithCancel(context.Background())

	// initialize components
	dbConnection := db.NewDatabaseConnection(dbCfg)
	if err = dbConnection.Migrate(); err != nil {
		glog.Errorf("Failed to migrate database: %v", err)
		os.Exit(1)
	}
	rateLimiter := email.NewRateLimiterService(dbConnection, cfg.LimitEmailPerTenant)

	cleanupWorker := workers.CleanupEmailSent{
		DbConn:       dbConnection,
		Period:       time.Second * time.Duration(cfg.EmailCleanupPeriodSeconds) * 60,
		ExpiredAfter: time.Hour * time.Duration(cfg.EmailCleanupExpiryDays) * 24,
	}
	go func() {
		err := cleanupWorker.Run(shutdownCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			glog.Errorf("failed to cleanup expired email events: %v", err)
		}
	}()

	emailSender, err := email.NewEmailSender(ctx, cfg, rateLimiter)
	if err != nil {
		glog.Errorf("Failed to initialise EmailSender implementation: %v", err)
		os.Exit(1)
	}

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

	cancelShutdownCtx()
	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("API Shutdown error: %v", err)
	}
	if err := metricServer.Close(); err != nil {
		glog.Errorf("closing metric server error: %v", err)
	}

	glog.Infof("Caught %s signal", sig)
	glog.Info("Email sender application has been stopped")
}
