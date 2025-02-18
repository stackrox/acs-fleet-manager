package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/email"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	apiErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// EmailHandler defines HTTP handlers for emailsender
type EmailHandler struct {
	emailSender email.Sender
}

// SendEmailRequest represents API requests for sending email
type SendEmailRequest struct {
	To         []string `json:"to"`
	RawMessage []byte   `json:"rawMessage"`
}

type Envelope map[string]interface{}

// NewEmailHandler ...
func NewEmailHandler(emailSender email.Sender) *EmailHandler {
	return &EmailHandler{
		emailSender: emailSender,
	}
}

// SendEmail is the HTTP handler function to send emails via emailsender
func (eh *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	var request SendEmailRequest

	jsonDecoder := json.NewDecoder(r.Body)
	jsonDecoder.DisallowUnknownFields()

	if err := jsonDecoder.Decode(&request); err != nil {
		shared.HandleError(r, w, apiErrors.MalformedRequest("failed to decode send email request payload"))
		return
	}

	claims, err := auth.GetClaimsFromContext(r.Context())
	if err != nil {
		shared.HandleError(r, w, apiErrors.Unauthenticated("failed to get token claims"))
		return
	}

	tenantID, err := claims.GetTenantID()
	if err != nil {
		shared.HandleError(r, w, apiErrors.Unauthenticated("failed to get tenantID"))
		return
	}

	if err := eh.emailSender.Send(r.Context(), request.To, request.RawMessage, tenantID); err != nil {
		var returnErr *apiErrors.ServiceError
		if errors.As(err, &email.RateLimitError{}) {
			returnErr = apiErrors.NewWithCause(apiErrors.ErrorTooManyRequests, err, "rate limitted")
		} else {
			returnErr = apiErrors.GeneralError("cannot send email")
		}
		shared.HandleError(r, w, returnErr)
		return
	}

	envelope := Envelope{
		"status": "sent",
	}
	if err := eh.jsonResponse(w, envelope, http.StatusOK); err != nil {
		glog.Errorf("Failed creating json response: %v", err)
		http.Error(w, "Cannot create json response", http.StatusInternalServerError)
	}
}

func (eh *EmailHandler) jsonResponse(w http.ResponseWriter, envelop Envelope, statusCode int) error {
	j, err := json.Marshal(envelop)
	if err != nil {
		return fmt.Errorf("failed to marshal: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err = w.Write(j)
	if err != nil {
		return fmt.Errorf("failed to write json response: %v", err)
	}

	return nil
}
