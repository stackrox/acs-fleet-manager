package server

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/sentry"
	"net"
	"net/http"
	"time"

	health "github.com/docker/go-healthcheck"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	updater = health.NewStatusUpdater()
)

var _ Server = &HealthCheckServer{}

type HealthCheckServer struct {
	httpServer        *http.Server
	serverConfig      *ServerConfig
	sentryTimeout     time.Duration
	healthCheckConfig *HealthCheckConfig
}

func NewHealthCheckServer(healthCheckConfig *HealthCheckConfig, serverConfig *ServerConfig, sentryConfig *sentry.Config) *HealthCheckServer {
	router := mux.NewRouter()
	health.DefaultRegistry = health.NewRegistry()
	health.Register("maintenance_status", updater)
	router.HandleFunc("/healthcheck", health.StatusHandler).Methods(http.MethodGet)
	router.HandleFunc("/healthcheck/down", downHandler).Methods(http.MethodPost)
	router.HandleFunc("/healthcheck/up", upHandler).Methods(http.MethodPost)

	srv := &http.Server{
		Handler: router,
		Addr:    healthCheckConfig.BindAddress,
	}

	return &HealthCheckServer{
		httpServer:        srv,
		serverConfig:      serverConfig,
		healthCheckConfig: healthCheckConfig,
		sentryTimeout:     sentryConfig.Timeout,
	}
}

func (s HealthCheckServer) Start() {
	go s.Run()
}

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

func (s HealthCheckServer) Stop() {
	err := s.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop health check server: %s", err)
	}
}

// Unimplemented
func (s HealthCheckServer) Listen() (listener net.Listener, err error) {
	return nil, nil
}

// Unimplemented
func (s HealthCheckServer) Serve(listener net.Listener) {
}

func upHandler(w http.ResponseWriter, r *http.Request) {
	updater.Update(nil)
}

func downHandler(w http.ResponseWriter, r *http.Request) {
	updater.Update(fmt.Errorf("maintenance mode"))
}
