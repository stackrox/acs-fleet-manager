package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig_Success(t *testing.T) {
	t.Setenv("FLEET_MANAGER_ENDPOINT", "http://127.0.0.1:8888")
	t.Setenv("AUTH_TYPE", "RHSSO")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "dummy")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "dummy")
	t.Setenv("HOSTNAME", "hostname-dummy")

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, cfg.FleetManagerEndpoint, "http://127.0.0.1:8888")
	assert.Equal(t, cfg.ProbePollPeriod, 5*time.Second)
	assert.Equal(t, cfg.ProbeName, "hostname-dummy")
}

func TestGetConfig_Failure(t *testing.T) {
	t.Setenv("AUTH_TYPE", "RHSSO")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "")

	_, err := GetConfig()

	assert.Error(t, err)
}
