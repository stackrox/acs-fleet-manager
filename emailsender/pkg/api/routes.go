package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/pkg/errors"

	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	acscsErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	loggingMiddleware "github.com/stackrox/acs-fleet-manager/pkg/server/logging"
)

// SetupRoutes configures API route mapping
func SetupRoutes(authConfig config.AuthConfig, emailHandler *EmailHandler) (http.Handler, error) {

	router := mux.NewRouter()

	// using a path prefix here to seperate endpoints that should use
	// middleware vs. endpoints that shouldn't for instance /health
	apiRouter := router.PathPrefix("/api").Subrouter()

	// add middlewares
	apiRouter.Use(
		loggingMiddleware.RequestLoggingMiddleware,
		EnsureJSONContentType,
		emailsenderAuthorizationMiddleware(authConfig),
	)

	router.HandleFunc("/health", HealthCheckHandler).Methods("GET")
	apiRouter.HandleFunc("/v1/acscsemail", emailHandler.SendEmail).Methods("POST")

	authLogger, err := sdk.NewGlogLoggerBuilder().
		InfoV(glog.Level(1)).
		DebugV(glog.Level(5)).
		Build()

	if err != nil {
		return nil, errors.Wrap(err, "failed to create auth logger")
	}

	authHandlerBuilder := authentication.NewHandler().
		Logger(authLogger).
		Error(fmt.Sprint(acscsErrors.ErrorUnauthenticated)).
		Service("ACSCS-EMAIL").
		Next(router).
		Public("/health")

	for _, keyURL := range authConfig.JwksURLs {
		authHandlerBuilder.KeysURL(keyURL)
	}

	authHandler, err := authHandlerBuilder.Build()

	if err != nil {
		return nil, errors.Wrap(err, "failed to create authentication handler")
	}

	return authHandler, nil
}
