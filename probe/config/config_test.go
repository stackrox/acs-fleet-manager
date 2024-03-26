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

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, cfg.FleetManagerEndpoint, "http://127.0.0.1:8888")
	assert.Equal(t, cfg.ProbePollPeriod, 5*time.Second)
}

func TestGetConfig_Failure(t *testing.T) {
	t.Setenv("AUTH_TYPE", "RHSSO")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "")

	cfg, err := GetConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestGetConfig_CentralSpec(t *testing.T) {
	t.Setenv("FLEET_MANAGER_ENDPOINT", "http://127.0.0.1:8888")
	t.Setenv("AUTH_TYPE", "RHSSO")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "dummy")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "dummy")
	t.Setenv("CENTRAL_SPECS", `[{ "cloudProvider": "aws", "region": "us-east-1" }, { "cloudProvider": "aws", "region": "eu-west-1" }]`)

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, CentralSpecs{
		{
			CloudProvider: "aws",
			Region:        "us-east-1",
		},
		{
			CloudProvider: "aws",
			Region:        "eu-west-1",
		},
	}, cfg.CentralSpecs)
}

func TestGetConfig_CentralSpecDefault(t *testing.T) {
	t.Setenv("FLEET_MANAGER_ENDPOINT", "http://127.0.0.1:8888")
	t.Setenv("AUTH_TYPE", "RHSSO")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "dummy")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "dummy")

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, CentralSpecs{
		{
			CloudProvider: "standalone",
			Region:        "standalone",
		},
	}, cfg.CentralSpecs)
}

func TestGetConfig_CentralSpecInvalidJson(t *testing.T) {
	t.Setenv("FLEET_MANAGER_ENDPOINT", "http://127.0.0.1:8888")
	t.Setenv("AUTH_TYPE", "RHSSO")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", "dummy")
	t.Setenv("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", "dummy")
	t.Setenv("CENTRAL_SPECS", `{ "cloudProvider": `)

	_, err := GetConfig()

	require.Error(t, err)
}
