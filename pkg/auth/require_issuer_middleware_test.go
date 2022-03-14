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

func TestRequireIssuerMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		token      *jwt.Token
		next       http.Handler
		errCode    errors.ServiceErrorCode
		wantIssuer []string
		wantCode   int
	}{
		{
			name: "should success when required issuer is set in JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "desiredIssuer",
				},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: []string{"desiredIssuer"},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name: "should success when issuer in the token matches one of the valid issuers",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "desiredIssuer2",
				},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: []string{"desiredIssuer", "desiredIssuer2", "desiredIssuer3"},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusOK,
		},
		{
			name: "should fail with the provided error when required issuer is different than the one set in the JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "anotherIssuer",
				},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: []string{"desiredIssuer"},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "should fail with the provided error when issuer field is not set in the JWT token",
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: []string{"desiredIssuer"},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "should fail when issuer in the token does not match any of the valid issuers",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "anotherIssuer",
				},
			},
			errCode:    errors.ErrorUnauthenticated,
			wantIssuer: []string{"desiredIssuer", "desiredIssuer2"},
			next: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}),
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			requireIssuerHandler := NewRequireIssuerMiddleware()
			toTest := setContextToken(requireIssuerHandler.RequireIssuer(tt.wantIssuer, tt.errCode)(tt.next), tt.token)
			req := httptest.NewRequest("GET", "http://example.com", nil)
			recorder := httptest.NewRecorder()
			toTest.ServeHTTP(recorder, req)
			gomega.Expect(recorder.Result().StatusCode).To(gomega.Equal(tt.wantCode))
		})
	}
}
