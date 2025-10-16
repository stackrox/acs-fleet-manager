// Package server ...
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/server/logging"

	"github.com/golang/glog"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
)

var _ environments.BootService = &APIServer{}

// APIServer ...
type APIServer struct {
	httpServer   *http.Server
	serverConfig *ServerConfig
}

// ServerOptions ...
type ServerOptions struct {
	di.Inject
	ServerConfig *ServerConfig
	IAMConfig    *iam.IAMConfig
	RouteLoaders []environments.RouteLoader
	Env          *environments.Env
}

// NewAPIServer ...
func NewAPIServer(options ServerOptions) *APIServer {
	s := &APIServer{
		httpServer:   nil,
		serverConfig: options.ServerConfig,
	}

	// mainRouter is top level "/"
	mainRouter := mux.NewRouter()
	mainRouter.NotFoundHandler = http.HandlerFunc(api.SendNotFound)
	mainRouter.MethodNotAllowedHandler = http.HandlerFunc(api.SendMethodNotAllowed)

	// Top-level middlewares

	// Operation ID middleware sets a relatively unique operation ID in the context of each request for debugging purposes
	mainRouter.Use(logger.OperationIDMiddleware)

	// Request logging middleware logs pertinent information about the request and response
	mainRouter.Use(logging.RequestLoggingMiddleware)

	for _, loader := range options.RouteLoaders {
		check(loader.AddRoutes(mainRouter), "error adding routes", 5*time.Second)
	}

	// referring to the router as type http.Handler allows us to add middleware via more handlers
	var mainHandler http.Handler = mainRouter

	var err error
	mainHandler, err = handlers.NewAuthenticationHandler(options.IAMConfig, mainHandler)
	check(err, "Unable to create authentication handler", 5*time.Second)

	mainHandler = gorillahandlers.CORS(
		gorillahandlers.AllowedMethods([]string{
			http.MethodDelete,
			http.MethodGet,
			http.MethodPatch,
			http.MethodPost,
			http.MethodPut,
		}),
		gorillahandlers.AllowedHeaders([]string{
			"Authorization",
			"Content-Type",
		}),
		gorillahandlers.MaxAge(int((10 * time.Minute).Seconds())),
	)(mainHandler)

	mainHandler = removeTrailingSlash(mainHandler)

	s.httpServer = &http.Server{
		Addr:    options.ServerConfig.BindAddress,
		Handler: mainHandler,
	}

	return s
}

// Serve start the blocking call to Serve.
// Useful for breaking up ListenAndServer (Start) when you require the server to be listening before continuing
func (s *APIServer) Serve(listener net.Listener) {
	var err error
	if s.serverConfig.EnableHTTPS {
		// Check https cert and key path
		if s.serverConfig.HTTPSCertFile == "" || s.serverConfig.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server",
				5*time.Second,
			)
		}

		// Serve with TLS
		glog.Infof("Serving with TLS at %s", s.serverConfig.BindAddress)
		err = s.httpServer.ServeTLS(listener, s.serverConfig.HTTPSCertFile, s.serverConfig.HTTPSKeyFile)
	} else {
		glog.Infof("Serving without TLS at %s", s.serverConfig.BindAddress)
		err = s.httpServer.Serve(listener)
	}

	// Web server terminated.
	check(err, "Web server terminated with errors", 5*time.Second)
	glog.Info("Web server terminated")
}

// Listen only starts the listener, not the server.
// Useful for breaking up ListenAndServer (Start) when you require the server to be listening before continuing
func (s *APIServer) listen() net.Listener {
	l, err := net.Listen("tcp", s.serverConfig.BindAddress)
	if err != nil {
		glog.Fatalf("Unable to start API server: %s", err)
	}
	return l
}

// Start starts listening on the configured port and start the server.
func (s *APIServer) Start() {
	listener := s.listen() // bind address in the same goroutine to avoid concurrency issues
	go s.Serve(listener)
}

// Stop stops the service
func (s *APIServer) Stop() {
	err := s.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop API server: %s", err)
	}
}
