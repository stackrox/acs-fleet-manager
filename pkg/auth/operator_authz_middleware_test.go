package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

func TestOperatorAuthzMiddleware_CheckClusterId(t *testing.T) {
	tests := []struct {
		name      string
		token     *jwt.Token
		clusterId string
		want      int
	}{
		{
			name: "should success when clusterId matches",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "12345",
			want:      http.StatusOK,
		},
		{
			name: "should fail when clusterId doesn't match",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "invalidid",
			want:      http.StatusNotFound,
		},
		{
			name: "should fail when clusterId claim isn't presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			clusterId: "12345",
			want:      http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// we need to use mux here to parse the id in the request url
			route := mux.NewRouter().PathPrefix("/agent-cluster/{id}").Subrouter()
			route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}).Methods(http.MethodGet)
			route.Use(func(handler http.Handler) http.Handler {
				return setContextToken(handler, tt.token)
			})
			route.Use(checkClusterId("id"))
			req := httptest.NewRequest("GET", "http://example.com/agent-cluster/"+tt.clusterId, nil)
			recorder := httptest.NewRecorder()
			route.ServeHTTP(recorder, req)
			status := recorder.Result().StatusCode
			if status != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, status)
			}
		})
	}
}

func TestOperatorAuthzMiddleware_CheckOCMToken(t *testing.T) {
	const JWKSEndpoint = "http://localhost"

	tests := []struct {
		name      string
		token     *jwt.Token
		clusterId string
		want      int
	}{
		{
			name: "should success when JWKS Endpoint matches",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":                            JWKSEndpoint,
					"fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "12345",
			want:      http.StatusOK,
		},
		{
			name: "should fail when JWKS iss claim is empty",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"iss":                            "",
					"fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "12345",
			want:      http.StatusNotFound,
		},
		{
			name: "should fail when JWKS iss claim is nil",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "12345",
			want:      http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// we need to use mux here to parse the id in the request url
			route := mux.NewRouter().PathPrefix("/agent-cluster/{id}").Subrouter()
			route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}).Methods(http.MethodGet)
			route.Use(func(handler http.Handler) http.Handler {
				return setContextToken(handler, tt.token)
			})
			requireIssuerMiddleware := NewRequireIssuerMiddleware()
			route.Use(requireIssuerMiddleware.RequireIssuer([]string{JWKSEndpoint}, errors.ErrorNotFound))
			req := httptest.NewRequest("GET", "http://example.com/agent-cluster/"+tt.clusterId, nil)
			recorder := httptest.NewRecorder()
			route.ServeHTTP(recorder, req)
			status := recorder.Result().StatusCode
			if status != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, status)
			}
		})
	}
}
