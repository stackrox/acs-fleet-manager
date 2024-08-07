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

	route.Use(CheckAllowedOrgIDs(allowedOrgIds))

	req := httptest.NewRequest("GET", "http://example.com/agent-clusters/1234", nil)
	recorder := httptest.NewRecorder()
	route.ServeHTTP(recorder, req)

	status := recorder.Result().StatusCode

	// We expect the 404 for unauthenticated access. This way we don't potentially leak the cluster ID to a client.
	assert.Equal(t, http.StatusNotFound, status)
}

func TestUseFleetShardAuthorizationMiddleware_DataPlaneOIDCIssuers(t *testing.T) {
	const validIssuer = "http://localhost"
	const kubernetesIssuer = "https://kubernetes.default.svc"
	validAudience := []string{"acs-fleet-manager-private-api"}
	validIssuers := []string{validIssuer}

	tests := map[string]struct {
		token                  *jwt.Token
		expectedStatusCode     int
		enableKubernetesIssuer bool
		dataplaneOIDCIssuers   []string
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
		"should succeed when kubernetes issuer enabled": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": kubernetesIssuer,
					"sub": "fleetshard-sync",
					"aud": []string{"acs-fleet-manager-private-api"},
				},
			},
			expectedStatusCode:     http.StatusOK,
			enableKubernetesIssuer: true,
		},
		"should succeed when kubernetes issuer enabled and no dataplane oidc issuers": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": kubernetesIssuer,
					"sub": "fleetshard-sync",
					"aud": []string{"acs-fleet-manager-private-api"},
				},
			},
			expectedStatusCode:     http.StatusOK,
			enableKubernetesIssuer: true,
			dataplaneOIDCIssuers:   []string{},
		},
		"should succeed when kubernetes issuer enabled and use dataplane oidc issuer": {
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss": validIssuer,
					"sub": "fleetshard-sync",
					"aud": []string{"acs-fleet-manager-private-api"},
				},
			},
			expectedStatusCode:     http.StatusOK,
			enableKubernetesIssuer: true,
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

			dataPlaneOIDCIssuers := validIssuers
			if tt.dataplaneOIDCIssuers != nil {
				dataPlaneOIDCIssuers = tt.dataplaneOIDCIssuers
			}

			UseFleetShardAuthorizationMiddleware(route,
				&iam.IAMConfig{
					RedhatSSORealm: &iam.IAMRealmConfig{
						ValidIssuerURI: "http://rhssorealm.local",
					},
					DataPlaneOIDCIssuers: &iam.OIDCIssuers{URIs: dataPlaneOIDCIssuers},
					KubernetesIssuer: &iam.KubernetesIssuer{
						Enabled:   tt.enableKubernetesIssuer,
						IssuerURI: kubernetesIssuer,
					},
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
