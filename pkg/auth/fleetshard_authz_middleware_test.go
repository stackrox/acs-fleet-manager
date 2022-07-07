package auth

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

func TestUseFleetShardAuthorizationMiddleware(t *testing.T) {
	const validIssuer = "http://localhost"

	tests := map[string]struct {
		token              *jwt.Token
		expectedStatusCode int
		allowedOrgIDs      AllowedOrgIDs
	}{
		"should succeed when org_id is contained within allowed org IDs": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    validIssuer,
					"org_id": "123",
				},
			},
			allowedOrgIDs:      AllowedOrgIDs{"123", "456"},
			expectedStatusCode: http.StatusOK,
		},
		"should fail when org_id is not contained within allowed org IDs": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    validIssuer,
					"org_id": "123",
				},
			},
			allowedOrgIDs:      AllowedOrgIDs{"456"},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when org_id is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			allowedOrgIDs:      AllowedOrgIDs{"456"},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when issuer cannot be verified": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":    "https://some-other-issuer",
					"org_id": "123",
				},
			},
			allowedOrgIDs:      AllowedOrgIDs{"123", "456"},
			expectedStatusCode: http.StatusNotFound,
		},
		"should fail when issuer can be verified but org_id is not set": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
				},
			},
			allowedOrgIDs:      AllowedOrgIDs{"123", "456"},
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

			UseFleetShardAuthorizationMiddleware(route, validIssuer, &FleetShardAuthZConfig{
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
	const validIssuer = "http://localhost"

	var allowedOrgIds = AllowedOrgIDs{"123", "345"}

	// Create the router but leave out the handler setting the context token.
	route := mux.NewRouter().PathPrefix("/agent-clusters/{id}").Subrouter()
	route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
		shared.WriteJSONResponse(writer, http.StatusOK, "")
	}).Methods(http.MethodGet)

	UseFleetShardAuthorizationMiddleware(route, validIssuer, &FleetShardAuthZConfig{AllowedOrgIDs: allowedOrgIds})

	req := httptest.NewRequest("GET", "http://example.com/agent-clusters/1234", nil)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, req)

	status := recorder.Result().StatusCode

	// We expect the 404 for unauthenticated access. This way we don't potentially leak the cluster ID to a client.
	assert.Equal(t, http.StatusNotFound, status)
}
