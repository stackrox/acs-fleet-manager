package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestSingleton_Success(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Cleanup(func() {
		_ = os.Unsetenv("CLUSTER_ID")
		cfg = nil
		cfgErr = nil
	})
	loadConfig()
	require.NoError(t, cfgErr)
	assert.Equal(t, cfg.FleetManagerEndpoint, "http://127.0.0.1:8000")
	assert.Equal(t, cfg.ClusterID, "some-value")
	assert.Equal(t, cfg.RuntimePollPeriod, 5*time.Second)
	assert.Equal(t, cfg.AuthType, "OCM")
	assert.Equal(t, cfg.RHSSOTokenFilePath, "/run/secrets/rhsso-token/token")
	assert.Empty(t, cfg.OCMRefreshToken)
}

func TestSingleton_Failure(t *testing.T) {
	t.Cleanup(func() {
		cfg = nil
		cfgErr = nil
	})
	loadConfig()
	assert.Error(t, cfgErr)
	assert.Nil(t, cfg)
}
