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
	acscsAPI "github.com/stackrox/acs-fleet-manager/pkg/api"
	acscsErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	acscsHandlers "github.com/stackrox/acs-fleet-manager/pkg/handlers"
	loggingMiddleware "github.com/stackrox/acs-fleet-manager/pkg/server/logging"
)

const emailsenderPrefix = "ACSCS-EMAIL"
const emailsenderErrorHREF = "/api/v1/acscsemail/errors/"

type authnHandlerBuilder func(router http.Handler, cfg config.AuthConfig) (http.Handler, error)

var _ authnHandlerBuilder = buildAuthnHandler

// SetupRoutes configures API route mapping
func SetupRoutes(authConfig config.AuthConfig, emailHandler *EmailHandler) (http.Handler, error) {
	return setupRoutes(buildAuthnHandler, authConfig, emailHandler)
}

func setupRoutes(authnHandlerFunc authnHandlerBuilder, authConfig config.AuthConfig, emailHandler *EmailHandler) (http.Handler, error) {
	router := mux.NewRouter()
	errorsHandler := acscsHandlers.NewErrorsHandler()

	router.NotFoundHandler = http.HandlerFunc(acscsAPI.SendNotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(acscsAPI.SendMethodNotAllowed)

	// using a path prefix here to seperate endpoints that should use
	// middleware vs. endpoints that shouldn't for instance /health
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

	// add middlewares
	apiV1Router.Use(
		loggingMiddleware.RequestLoggingMiddleware,
		EnsureJSONContentType,
		// this middleware is supposed to validate if the client is authorized to do the desired request
		// as opposed to the authnHandler which authenticates a requests with a token and stores claims in
		// the requests context
		emailsenderAuthorizationMiddleware(authConfig),
	)

	// health endpoint
	router.HandleFunc("/health", HealthCheckHandler).Methods("GET")

	// errors endpoint
	router.HandleFunc("/api/v1/acscsemail/errors/{id}", errorsHandler.Get).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/acscsemail/errors", errorsHandler.List).Methods(http.MethodGet)

	// send email endpoint
	apiV1Router.HandleFunc("/acscsemail", emailHandler.SendEmail).Methods("POST")

	// this settings are to make sure the middlewares shared with acs-fleet-manager
	// print a prefix and href matching to the emailsender application
	acscsErrors.ErrorCodePrefixOverride = emailsenderPrefix
	acscsErrors.ErrorHREFOverride = emailsenderErrorHREF

	return authnHandlerFunc(router, authConfig)
}

func buildAuthnHandler(router http.Handler, cfg config.AuthConfig) (http.Handler, error) {
	authnLogger, err := sdk.NewGlogLoggerBuilder().
		InfoV(glog.Level(1)).
		DebugV(glog.Level(5)).
		Build()

	if err != nil {
		return nil, errors.Wrap(err, "failed to create auth logger")
	}

	authnHandlerBuilder := authentication.NewHandler().
		Logger(authnLogger).
		Error(fmt.Sprint(acscsErrors.ErrorUnauthenticated)).
		Service(emailsenderPrefix).
		Next(router).
		Public("/health").
		Public("/api/v1/acscsemail/errors/?[0-9]*")

	for _, keyURL := range cfg.JwksURLs {
		authnHandlerBuilder.KeysURL(keyURL)
	}

	for _, keyFile := range cfg.JwksFiles {
		authnHandlerBuilder.KeysFile(keyFile)
	}

	authHandler, err := authnHandlerBuilder.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create authentication handler")
	}

	return authHandler, nil
}
