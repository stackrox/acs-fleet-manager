package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleton_Success(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Cleanup(func() {
		_ = os.Unsetenv("CLUSTER_ID")
	})
	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.FleetManagerEndpoint, "http://127.0.0.1:8000")
	assert.Equal(t, cfg.ClusterID, "some-value")
	assert.Equal(t, cfg.RuntimePollPeriod, 5*time.Second)
	assert.Equal(t, cfg.AuthType, "OCM")
	assert.Equal(t, cfg.RHSSOTokenFilePath, "/run/secrets/rhsso-token/token")
	assert.Empty(t, cfg.OCMRefreshToken)
}

func TestSingleton_Failure(t *testing.T) {
	t.Cleanup(func() {
	})
	cfg, err := GetConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
