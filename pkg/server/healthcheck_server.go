package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sentry"

	health "github.com/docker/go-healthcheck"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	updater = health.NewStatusUpdater()
)

var _ Server = &HealthCheckServer{}

// HealthCheckServer ...
type HealthCheckServer struct {
	httpServer          *http.Server
	serverConfig        *ServerConfig
	sentryTimeout       time.Duration
	healthCheckConfig   *HealthCheckConfig
	dbConnectionFactory *db.ConnectionFactory
}

// NewHealthCheckServer ...
func NewHealthCheckServer(healthCheckConfig *HealthCheckConfig, serverConfig *ServerConfig, sentryConfig *sentry.Config, dbConnectionFactory *db.ConnectionFactory) *HealthCheckServer {
	router := mux.NewRouter()
	health.DefaultRegistry = health.NewRegistry()
	health.Register("maintenance_status", updater)

	srv := &http.Server{
		Handler: router,
		Addr:    healthCheckConfig.BindAddress,
	}

	healthServer := &HealthCheckServer{
		httpServer:          srv,
		serverConfig:        serverConfig,
		healthCheckConfig:   healthCheckConfig,
		sentryTimeout:       sentryConfig.Timeout,
		dbConnectionFactory: dbConnection,
	}

	router.HandleFunc("/healthcheck", health.StatusHandler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck/down", downHandler).Methods(http.MethodPost)
	router.HandleFunc("/healthcheck/up", upHandler).Methods(http.MethodPost)
	router.HandleFunc("/healthcheck/ready", healthServer.ready).Methods(http.MethodGet)

	return healthServer
}

// Start ...
func (s HealthCheckServer) Start() {
	go s.Run()
}

// Run ...
func (s HealthCheckServer) Run() {
	var err error
	if s.healthCheckConfig.EnableHTTPS {
		if s.serverConfig.HTTPSCertFile == "" || s.serverConfig.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server", s.sentryTimeout,
			)
		}

		// Serve with TLS
		glog.Infof("Serving HealthCheck with TLS at %s", s.healthCheckConfig.BindAddress)
		err = s.httpServer.ListenAndServeTLS(s.serverConfig.HTTPSCertFile, s.serverConfig.HTTPSKeyFile)
	} else {
		glog.Infof("Serving HealthCheck without TLS at %s", s.healthCheckConfig.BindAddress)
		err = s.httpServer.ListenAndServe()
	}
	check(err, "HealthCheck server terminated with errors", s.sentryTimeout)
	glog.Infof("HealthCheck server terminated")
}

// Stop ...
func (s HealthCheckServer) Stop() {
	err := s.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop health check server: %s", err)
	}
}

// Listen Unimplemented
func (s HealthCheckServer) Listen() (listener net.Listener, err error) {
	return nil, nil
}

// Serve Unimplemented
func (s HealthCheckServer) Serve(listener net.Listener) {
}

func upHandler(w http.ResponseWriter, r *http.Request) {
	updater.Update(nil)
}

func downHandler(w http.ResponseWriter, r *http.Request) {
	updater.Update(fmt.Errorf("maintenance mode"))
}

// ready checks for the service dependencies such as DB connection.
// A "ready" service means it is prepared to serve traffic. It is used by the readinessProbe.
func (s HealthCheckServer) ready(w http.ResponseWriter, r *http.Request) {
	err := s.dbConnectionFactory.CheckConnection()
	if err != nil {
		api.SendServiceUnavailable(w, r, "DB connection failed")
	}
}
