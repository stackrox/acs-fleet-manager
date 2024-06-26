package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/email"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

type EmailHandler struct {
	emailSender email.Sender
}

// SendEmailRequest represents API requests for sending email
type SendEmailRequest struct {
	To         []string `json:"to"`
	RawMessage []byte   `json:"rawMessage"`
}

type Envelope map[string]interface{}

func NewEmailHandler(emailSender email.Sender) *EmailHandler {
	return &EmailHandler{
		emailSender: emailSender,
	}
}

func (eh *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	var request SendEmailRequest

	jsonDecoder := json.NewDecoder(r.Body)
	jsonDecoder.DisallowUnknownFields()

	if err := jsonDecoder.Decode(&request); err != nil {
		eh.errorResponse(w, "Cannot decode send email request payload", http.StatusBadRequest)
		return
	}

	claims, err := auth.GetClaimsFromContext(r.Context())
	if err != nil {
		shared.HandleError(r, w, errors.Unauthenticated("failed to get token claims"))
		return
	}

	tenantID, err := claims.GetTenantID()
	if err != nil {
		shared.HandleError(r, w, errors.Unauthenticated("failed to get tenantID"))
		return
	}

	if err := eh.emailSender.Send(r.Context(), request.To, request.RawMessage, tenantID); err != nil {
		eh.errorResponse(w, "Cannot send email", http.StatusInternalServerError)
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

func (eh *EmailHandler) errorResponse(w http.ResponseWriter, message string, statusCode int) {
	envelope := Envelope{
		"error": message,
	}

	if err := eh.jsonResponse(w, envelope, statusCode); err != nil {
		glog.Errorf("Failed creating error json response: %v", err)
		http.Error(w, "Can not create error json response", http.StatusInternalServerError)
	}
}
