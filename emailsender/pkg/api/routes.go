package api

import (
	"github.com/gorilla/mux"
)

// SetupRoutes configures API route mapping
func SetupRoutes(router *mux.Router) {
	// add middlewares
	router.Use(LogRequest, EnsureJSONContentType)

	router.HandleFunc("/health", HealthCheckHandler).Methods("GET")
	router.HandleFunc("/v1/acscsemail", SendEmailHandler).Methods("POST")
}
