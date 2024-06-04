package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/email"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockEmailSender struct {
	SendFunc func(ctx context.Context, to []string, rawMessage []byte) error
}

func (m *MockEmailSender) Send(ctx context.Context, to []string, rawMessage []byte) error {
	return m.SendFunc(ctx, to, rawMessage)
}

var simpleEmailSender = &MockEmailSender{
	SendFunc: func(ctx context.Context, to []string, rawMessage []byte) error {
		return nil
	},
}

func TestEmailHandler_SendEmail(t *testing.T) {
	subject := "Test subject"
	textBody := "text body"
	var messageBuf bytes.Buffer
	messageBuf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	messageBuf.WriteString(textBody)
	rawMessage := messageBuf.Bytes()

	sendEmailRequest := SendEmailRequest{
		To:         []string{"to1@example.com", "to2@example.com"},
		RawMessage: rawMessage,
	}
	jsonReq, _ := json.Marshal(sendEmailRequest)
	invalidJsonReq, _ := json.Marshal(map[string]string{
		"invalid": "JSON",
	})

	tests := []struct {
		name        string
		emailSender email.Sender
		req         *http.Request
		wantCode    int
		wantBody    string
	}{
		{
			name:        "should return JSON response with StatusOK to a valid email request",
			emailSender: simpleEmailSender,
			req:         httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonReq)),
			wantCode:    http.StatusOK,
			wantBody:    `{"status":"sent"}`,
		},
		{
			name:        "should return JSON error with StatusBadRequest when cannot decode request",
			emailSender: simpleEmailSender,
			req:         httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(invalidJsonReq)),
			wantCode:    http.StatusBadRequest,
			wantBody:    `{"error":"Cannot decode send email request payload"}`,
		},
		{
			name: "should return JSON error with StatusInternalServerError when cannot send email",
			emailSender: &MockEmailSender{
				SendFunc: func(ctx context.Context, to []string, rawMessage []byte) error {
					return errors.New("failed to send email")
				},
			},
			req:      httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonReq)),
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"Cannot send email"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := &EmailHandler{
				emailSender: tt.emailSender,
			}
			resp := httptest.NewRecorder()
			eh.SendEmail(resp, tt.req)

			if resp.Result().StatusCode != tt.wantCode {
				t.Errorf("expected status code %d, got %d", tt.wantCode, resp.Result().StatusCode)
			}

			if resp.Body.String() != tt.wantBody {
				t.Errorf("expected body %s, got %s", tt.wantBody, resp.Body.String())
			}
		})
	}
}
