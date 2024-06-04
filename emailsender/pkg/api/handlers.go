// Package api handlers contains HTTP request -> HTTP response functions
package api

import (
	"net/http"
)

// HealthCheckHandler returns 200 HTTP status code
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
