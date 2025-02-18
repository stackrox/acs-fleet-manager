package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/email"
)

type MockEmailSender struct {
	SendFunc func(ctx context.Context, to []string, rawMessage []byte) error
}

func (m *MockEmailSender) Send(ctx context.Context, to []string, rawMessage []byte, tenantID string) error {
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

	defaultToken := &jwt.Token{
		Claims: jwt.MapClaims{
			"iss":    "https://sso.redhat.com/auth/realms/redhat-external",
			"aud":    "test-audience",
			"sub":    "test-sub",
			"org_id": "test-org",
		},
	}

	tests := []struct {
		name            string
		emailSender     email.Sender
		req             *http.Request
		wantCode        int
		wantBody        string
		wantErrorReason string
	}{
		{
			name:        "should return JSON response with StatusOK to a valid email request",
			emailSender: simpleEmailSender,
			req:         httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonReq)),
			wantCode:    http.StatusOK,
			wantBody:    `{"status":"sent"}`,
		},
		{
			name:            "should return JSON error with StatusBadRequest when cannot decode request",
			emailSender:     simpleEmailSender,
			req:             httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(invalidJsonReq)),
			wantCode:        http.StatusBadRequest,
			wantErrorReason: "failed to decode send email request payload",
		},
		{
			name: "should return JSON error with StatusInternalServerError when cannot send email",
			emailSender: &MockEmailSender{
				SendFunc: func(ctx context.Context, to []string, rawMessage []byte) error {
					return errors.New("failed to send email")
				},
			},
			req:             httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonReq)),
			wantCode:        http.StatusInternalServerError,
			wantErrorReason: "cannot send email",
		},
		{
			name: "should return 429 status when SendFunc returns RateLimitError",
			emailSender: &MockEmailSender{
				SendFunc: func(ctx context.Context, to []string, rawMessage []byte) error {
					return email.RateLimitError{TenantID: "test-sub"}
				},
			},
			req:             httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(jsonReq)),
			wantCode:        http.StatusTooManyRequests,
			wantErrorReason: "rate limitted",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eh := &EmailHandler{
				emailSender: tt.emailSender,
			}
			resp := httptest.NewRecorder()

			ctx := authentication.ContextWithToken(tt.req.Context(), defaultToken)
			req := tt.req.WithContext(ctx)
			eh.SendEmail(resp, req)

			if resp.Result().StatusCode != tt.wantCode {
				t.Errorf("expected status code %d, got %d", tt.wantCode, resp.Result().StatusCode)
			}

			if tt.wantErrorReason != "" {
				var respDecoded map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&respDecoded); err != nil {
					t.Errorf("failed to decoded response body")
				}
				errorReason, ok := respDecoded["reason"]
				if !ok {
					t.Errorf("response error body does not have reason key")
				}
				if errorReason != tt.wantErrorReason {
					t.Errorf("expected error reason %s, got %s", tt.wantBody, errorReason)
				}
			}

			if tt.wantBody != "" && resp.Body.String() != tt.wantBody {
				t.Errorf("expected body %s, got %s", tt.wantBody, resp.Body.String())
			}
		})
	}
}
