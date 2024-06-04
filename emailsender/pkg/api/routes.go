package api

import (
	"github.com/gorilla/mux"
	loggingMiddleware "github.com/stackrox/acs-fleet-manager/pkg/server/logging"
)

// SetupRoutes configures API route mapping
func SetupRoutes(router *mux.Router, emailHandler *EmailHandler) {
	// add middlewares
	router.Use(loggingMiddleware.RequestLoggingMiddleware, EnsureJSONContentType)

	router.HandleFunc("/health", HealthCheckHandler).Methods("GET")
	router.HandleFunc("/api/v1/acscsemail", emailHandler.SendEmail).Methods("POST")
}
