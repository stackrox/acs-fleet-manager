package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stretchr/testify/assert"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

func TestUseFleetShardAuthorizationMiddleware(t *testing.T) {
	const validIssuer = "http://localhost"

	tests := map[string]struct {
		token              *jwt.Token
		expectedStatusCode int
		allowedOrgIDs      ClaimValues
	}{
		"should succeed when org_id is contained within allowed org IDs": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    validIssuer,
					"org_id": "123",
				},
			},
			allowedOrgIDs:      ClaimValues{"123", "456"},
			expectedStatusCode: http.StatusOK,
		},
		"should fail when org_id is not contained within allowed org IDs": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    validIssuer,
					"org_id": "123",
				},
			},
			allowedOrgIDs:      ClaimValues{"456"},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when org_id is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			allowedOrgIDs:      ClaimValues{"456"},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when issuer cannot be verified": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    "https://some-other-issuer",
					"org_id": "123",
				},
			},
			allowedOrgIDs:      ClaimValues{"123", "456"},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when issuer can be verified but org_id is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
				},
			},
			allowedOrgIDs:      ClaimValues{"123", "456"},
			expectedStatusCode: http.StatusNotFound,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			route := mux.NewRouter().PathPrefix("/agent-clusters/{id}").Subrouter()
			route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}).Methods(http.MethodGet)
			route.Use(func(handler http.Handler) http.Handler {
				return setContextToken(handler, tt.token)
			})

			UseFleetShardAuthorizationMiddleware(route,
				&iam.IAMConfig{
					RedhatSSORealm: &iam.IAMRealmConfig{
						ValidIssuerURI: validIssuer,
					},
					DataPlaneOIDCIssuers: &iam.OIDCIssuers{},
				},
				&FleetShardAuthZConfig{
					AllowedOrgIDs: tt.allowedOrgIDs,
				})

			req := httptest.NewRequest("GET", "http://example.com/agent-clusters/1234", nil)
			recorder := httptest.NewRecorder()
			route.ServeHTTP(recorder, req)

			status := recorder.Result().StatusCode
			assert.Equal(t, tt.expectedStatusCode, status)
		})
	}
}

func TestUseFleetShardAuthorizationMiddleware_NoTokenSet(t *testing.T) {
	var allowedOrgIds = ClaimValues{"123", "345"}

	// Create the router but leave out the handler setting the context token.
	route := mux.NewRouter().PathPrefix("/agent-clusters/{id}").Subrouter()
	route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
		shared.WriteJSONResponse(writer, http.StatusOK, "")
	}).Methods(http.MethodGet)
	route.Use(func(handler http.Handler) http.Handler {
		return setContextToken(handler, nil)
	})

	route.Use(checkAllowedOrgIDs(allowedOrgIds))

	req := httptest.NewRequest("GET", "http://example.com/agent-clusters/1234", nil)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, req)

	status := recorder.Result().StatusCode

	// We expect the 404 for unauthenticated access. This way we don't potentially leak the cluster ID to a client.
	assert.Equal(t, http.StatusNotFound, status)
}

func TestUseFleetShardAuthorizationMiddleware_DataPlaneOIDCIssuers(t *testing.T) {
	const validIssuer = "http://localhost"
	validAudience := []string{"acs-fleet-manager-private-api"}

	tests := map[string]struct {
		token              *jwt.Token
		expectedStatusCode int
	}{
		"should succeed when sub is equal the allowed subject": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"sub": "fleetshard-sync",
					"aud": validAudience,
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		"should fail when sub is not equal the allowed subject": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"sub": "third-party-service",
					"aud": "acs-fleet-manager-private-api",
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when sub is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when issuer cannot be verified": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    "https://some-other-issuer",
					"org_id": "123",
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when issuer can be verified but sub is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"aud": validAudience,
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when audience is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"sub": "fleetshard-sync",
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when audience is not allowed": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"sub": "fleetshard-sync",
					"aud": []string{"https://kubernetes.default.svc"},
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		"should succeed when at least one audience is allowed": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"sub": "fleetshard-sync",
					"aud": []string{"other-api", "acs-fleet-manager-private-api"},
				},
			},
			expectedStatusCode: http.StatusOK,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			route := mux.NewRouter().PathPrefix("/agent-clusters/{id}").Subrouter()
			route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}).Methods(http.MethodGet)
			route.Use(func(handler http.Handler) http.Handler {
				return setContextToken(handler, tt.token)
			})

			UseFleetShardAuthorizationMiddleware(route,
				&iam.IAMConfig{
					RedhatSSORealm: &iam.IAMRealmConfig{
						ValidIssuerURI: "http://rhssorealm.local",
					},
					DataPlaneOIDCIssuers: &iam.OIDCIssuers{URIs: []string{validIssuer}},
				},
				&FleetShardAuthZConfig{
					AllowedSubjects:  []string{"fleetshard-sync"},
					AllowedAudiences: validAudience,
				})

			req := httptest.NewRequest("GET", "http://example.com/agent-clusters/1234", nil)
			recorder := httptest.NewRecorder()
			route.ServeHTTP(recorder, req)

			status := recorder.Result().StatusCode
			assert.Equal(t, tt.expectedStatusCode, status)
		})
	}
}
