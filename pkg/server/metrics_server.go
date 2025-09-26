package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"

	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
)

var _ environments.BootService = &MetricsServer{}

// NewMetricsServer ...
func NewMetricsServer(metricsConfig *MetricsConfig, serverConfig *ServerConfig) *MetricsServer {
	mainRouter := mux.NewRouter()
	mainRouter.NotFoundHandler = http.HandlerFunc(api.SendNotFound)

	// metrics endpoint
	prometheusMetricsHandler := handlers.NewPrometheusMetricsHandler()
	mainRouter.Handle("/metrics", prometheusMetricsHandler.Handler())

	var mainHandler http.Handler = mainRouter

	s := &MetricsServer{
		serverConfig:  serverConfig,
		metricsConfig: metricsConfig,
	}
	s.httpServer = &http.Server{
		Addr:    metricsConfig.BindAddress,
		Handler: mainHandler,
	}
	return s
}

// MetricsServer ...
type MetricsServer struct {
	httpServer    *http.Server
	serverConfig  *ServerConfig
	metricsConfig *MetricsConfig
}

// Start ...
func (s MetricsServer) Start() {
	go s.run()
}

func (s MetricsServer) run() {
	glog.Infof("start metrics server")
	var err error
	if s.metricsConfig.EnableHTTPS {
		if s.serverConfig.HTTPSCertFile == "" || s.serverConfig.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server", 5*time.Second,
			)
		}

		// Serve with TLS
		glog.Infof("Serving Metrics with TLS at %s", s.serverConfig.BindAddress)
		err = s.httpServer.ListenAndServeTLS(s.serverConfig.HTTPSCertFile, s.serverConfig.HTTPSKeyFile)
	} else {
		glog.Infof("Serving Metrics without TLS at %s", s.metricsConfig.BindAddress)
		err = s.httpServer.ListenAndServe()
	}
	check(err, "Metrics server terminated with errors", 5*time.Second)
	glog.Infof("Metrics server terminated")
}

// Stop ...
func (s MetricsServer) Stop() {
	err := s.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop health check server: %s", err)
	}
}
