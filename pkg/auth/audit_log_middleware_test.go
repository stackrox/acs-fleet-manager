package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
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
			name: "should success",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			auditLogMW := NewAuditLogMiddleware()
			toTest := setContextToken(auditLogMW.AuditLog(tt.errCode)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Result().StatusCode).To(gomega.Equal(tt.wantCode))
		})
	}
}
