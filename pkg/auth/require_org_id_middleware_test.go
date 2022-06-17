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

func TestRequireOrgIDMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		token    *jwt.Token
		next     http.Handler
		errCode  errors.ServiceErrorCode
		wantCode int
	}{
		{
			name: "should success when org_id claim is set in JWT token and it is not empty",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					tenantIdClaim: "correct_org_id",
				},
			},
			errCode: errors.ErrorUnauthenticated,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name: "should fail when org_id claim is not set in JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"anotherclaimkey": "anotherclaimvalue",
				},
			},
			errCode: errors.ErrorUnauthenticated,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "should fail when org_id claim is set in JWT token but it is empty",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					tenantIdClaim: "",
				},
			},
			errCode: errors.ErrorUnauthenticated,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "should fail when org_id claim is set in JWT token but it is a non-string type",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					tenantIdClaim: 3,
				},
			},
			errCode: errors.ErrorUnauthenticated,
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireIssuerHandler := NewRequireOrgIDMiddleware()
			toTest := setContextToken(requireIssuerHandler.RequireOrgID(tt.errCode)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			Expect(recorder.Result().StatusCode).To(Equal(tt.wantCode))
		})
	}
}
