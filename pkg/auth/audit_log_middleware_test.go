package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

func TestAuditLogMiddleware_AuditLog(t *testing.T) {
	tests := []struct {
		name     string
		token    *jwt.Token
		next     http.Handler
		errCode  errors.ServiceErrorCode
		wantCode int
	}{
		{
			name: "should pass for tenant Username",
			token: &jwt.Token{Claims: jwt.MapClaims{
				"username": "test user",
			}},
			errCode: errors.ErrorBadRequest,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name: "should pass for ssoUsernameKey",
			token: &jwt.Token{Claims: jwt.MapClaims{
				"preferred_username": "test user",
			}},
			errCode: errors.ErrorBadRequest,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditLogMW := NewAuditLogMiddleware()
			toTest := setContextToken(auditLogMW.AuditLog(tt.errCode)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			Expect(recorder.Result().StatusCode).To(Equal(tt.wantCode))
		})
	}
}
