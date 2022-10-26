package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig_Success(t *testing.T) {
	t.Setenv("FLEET_MANAGER_ENDPOINT", "http://127.0.0.1:8888")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "dummy")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "dummy")

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, cfg.FleetManagerEndpoint, "http://127.0.0.1:8888")
	assert.Equal(t, cfg.RuntimePollPeriod, 5*time.Second)
}

func TestGetConfig_Failure(t *testing.T) {
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}
