// Package api handlers contains HTTP request -> HTTP response functions
package api

import (
	"net/http"
)

// Represents Emailsender openAPI definition
type openAPIHandler struct {
	OpenAPIDefinition string
}

// NewOpenAPIHandler ...
func NewOpenAPIHandler(openAPIDefinition string) *openAPIHandler {
	return &openAPIHandler{openAPIDefinition}
}

// HealthCheckHandler returns 200 HTTP status code
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h openAPIHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h.OpenAPIDefinition))
}
