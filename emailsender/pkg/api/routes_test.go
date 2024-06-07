package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/emailsender/config"
	"github.com/stretchr/testify/require"
)

func TestHealthSuccessWithNoAuth(t *testing.T) {
	handler, err := SetupRoutes(config.AuthConfig{
		JwksURLs: []string{"test-key-url.test"},
	}, nil)
	require.NoError(t, err, "failed to setup router")

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	require.NoError(t, err, "failed to create request")

	handler.ServeHTTP(rec, req)

	require.Equal(t, 200, rec.Code)
}

func TestAuthnHandlerIsUsed(t *testing.T) {
	handler, err := setupRoutes(buildAlwaysDenyAuthnHandler, config.AuthConfig{}, nil)
	require.NoError(t, err, "failed to setup router")

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/test", nil)
	require.NoError(t, err, "failed to create request")

	handler.ServeHTTP(rec, req)

	require.Equal(t, 401, rec.Code)
}

func buildAlwaysDenyAuthnHandler(router http.Handler, cfg config.AuthConfig) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}), nil
}
