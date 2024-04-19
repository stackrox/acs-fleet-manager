package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigSuccess(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("SERVER_ADDRESS", ":8888")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("HTTPS_CERT_FILE", "/some/tls.crt")
	t.Setenv("HTTPS_KEY_FILE", "/some/tls.key")
	t.Setenv("METRICS_ADDRESS", ":9999")

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, cfg.ClusterID, "test-1")
	assert.Equal(t, cfg.ServerAddress, ":8888")
	assert.Equal(t, cfg.EnableHTTPS, true)
	assert.Equal(t, cfg.HTTPSCertFile, "/some/tls.crt")
	assert.Equal(t, cfg.HTTPSKeyFile, "/some/tls.key")
}

func TestGetConfigFailureMissingClusterID(t *testing.T) {
	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfigFailureEnabledHTTPSMissingCert(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("HTTPS_KEY_FILE", "/some/tls.key")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfigFailureEnabledHTTPSMissingKey(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("ENABLE_HTTPS", "true")
	t.Setenv("HTTPS_CERT_FILE", "/some/tls.crt")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfigFailureEnabledHTTPSOnly(t *testing.T) {
	t.Setenv("CLUSTER_ID", "test-1")
	t.Setenv("ENABLE_HTTPS", "true")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}
