package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var called bool
var nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	called = true
})
var testHandler = EnsureJSONContentType(nextHandler)
var req = httptest.NewRequest("GET", "http://testing", nil)

func TestEnsureJSONContentTypeEmptyHeader(t *testing.T) {
	called = false
	testHandler.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, called)
}

func TestEnsureJSONContentTypeValid(t *testing.T) {
	called = false
	req.Header.Set("Content-Type", "application/json")
	testHandler.ServeHTTP(httptest.NewRecorder(), req)

	assert.True(t, called)
}

func TestEnsureJSONContentTypeInvalid(t *testing.T) {
	called = false
	req.Header.Set("Content-Type", "broken")
	testHandler.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, called)
}

func TestAuthorizationMiddleware(t *testing.T) {

	tests := map[string]struct {
		cfg          config.AuthConfig
		token        *jwt.Token
		expectedCode int
	}{
		"unauthorized if issuer does not match": {
			cfg: config.AuthConfig{AllowedIssuer: []string{"test-issuer"}},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "invalid-issuer",
				},
			},
			expectedCode: 403,
		},
		"unauthorized if audience does not match": {
			cfg: config.AuthConfig{
				AllowedIssuer:    []string{"test-issuer"},
				AllowedAudiences: []string{"test-audience"},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "my-test-issuer",
					"aud": "invalid-audience",
				},
			},
			expectedCode: 403,
		},
		"unauthorized if subject does not match": {
			cfg: config.AuthConfig{
				AllowedIssuer:    []string{"test-issuer"},
				AllowedAudiences: []string{"test-audience"},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "test-issuer",
					"aud": "test-audience",
					"sub": "does not match central SA regexp",
				},
			},
			expectedCode: 403,
		},
		"authorized if not ocm and expected claims match": {
			cfg: config.AuthConfig{
				AllowedIssuer:    []string{"test-issuer"},
				AllowedAudiences: []string{"test-audience"},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": "test-issuer",
					"aud": "test-audience",
					"sub": "system:serviceaccount:rhacs-abc:central",
				},
			},
			expectedCode: 200,
		},
		"authorized if not ocm and org_id does not not match": {
			cfg: config.AuthConfig{
				AllowedIssuer:    []string{"test-issuer"},
				AllowedAudiences: []string{"test-audience"},
				AllowedOrgIDs:    []string{"test-org"},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    "test-issuer",
					"aud":    "test-audience",
					"sub":    "system:serviceaccount:rhacs-abc:central",
					"org_id": "invalid-org",
				},
			},
			expectedCode: 200,
		},
		"not found if ocm and org ID does not match": {
			// this returns 404 since the given org_id authorization method from fleet-manager return 404 in
			// cases where the org is not authorized
			cfg: config.AuthConfig{
				AllowedIssuer:    []string{"https://sso.redhat.com/auth/realms/redhat-external"},
				AllowedAudiences: []string{"test-audience"},
				AllowedOrgIDs:    []string{"test-org"},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    "https://sso.redhat.com/auth/realms/redhat-external",
					"aud":    "test-audience",
					"sub":    "system:serviceaccount:rhacs-abc:central",
					"org_id": "invalid-org",
				},
			},
			expectedCode: 404,
		},
		"authorized if ocm and expected claims match": {
			cfg: config.AuthConfig{
				AllowedIssuer:    []string{"https://sso.redhat.com/auth/realms/redhat-external"},
				AllowedAudiences: []string{"test-audience"},
				AllowedOrgIDs:    []string{"test-org"},
			},
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    "https://sso.redhat.com/auth/realms/redhat-external",
					"aud":    "test-audience",
					"sub":    "arbitrary-sub",
					"org_id": "test-org",
				},
			},
			expectedCode: 200,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			req, err := http.NewRequest(http.MethodGet, "/api/v1/test", nil)
			require.NoError(t, err, "failed to create HTTP req")
			ctx := authentication.ContextWithToken(req.Context(), tc.token)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			authzHandler := emailsenderAuthorizationMiddleware(tc.cfg)
			authzHandler.Middleware(successHandler()).ServeHTTP(rec, req)

			require.Equal(t, tc.expectedCode, rec.Result().StatusCode)
		})
	}
}

func successHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
}
