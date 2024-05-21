// Package api handlers contains HTTP request -> HTTP response functions
package api

import (
	"encoding/json"
	"net/http"
)

// EmailRequest represents API requests for sending email
type EmailRequest struct {
	Recipient string
	Body      string
}

// SendEmailHandler handles sending email API endpoint
func SendEmailHandler(w http.ResponseWriter, r *http.Request) {
	var email EmailRequest

	if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
		http.Error(w, "Can not decode request payload", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HealthCheckHandler returns 200 HTTP status code
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
