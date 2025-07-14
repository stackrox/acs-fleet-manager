package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleton_Success(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.FleetManagerEndpoint, "http://127.0.0.1:8000")
	assert.Equal(t, cfg.ClusterID, "some-value")
	assert.Equal(t, cfg.RuntimePollPeriod, 5*time.Second)
	assert.Equal(t, cfg.AuthType, "SERVICE_ACCOUNT_TOKEN")
}

func TestSingleton_Failure(t *testing.T) {
	cfg, err := GetConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestSingleton_Success_WhenManagedDBEnabled(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Setenv("MANAGED_DB_ENABLED", "true")
	t.Setenv("MANAGED_DB_SECURITY_GROUP", "some-group")
	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.ManagedDB.Enabled, true)
	assert.Equal(t, cfg.ManagedDB.SecurityGroup, "some-group")
}

func TestSingleton_Failure_WhenManagedDBEnabledAndManagedDbSecurityGroupNotSet(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Setenv("MANAGED_DB_ENABLED", "true")
	cfg, err := GetConfig()
	assert.Error(t, err, "MANAGED_DB_ENABLED == true and MANAGED_DB_SECURITY_GROUP unset in the environment")
	assert.Nil(t, cfg)
}

func TestSingleton_ManagedDBTags(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Setenv("MANAGED_DB_TAGS_0_KEY", "DataplaneClusterName")
	t.Setenv("MANAGED_DB_TAGS_0_VALUE", "acs-dev-dp-01")

	cfg, err := GetConfig()
	require.NoError(t, err)
	require.Len(t, cfg.ManagedDB.Tags, 1)
	assert.Equal(t, cfg.ManagedDB.Tags[0].Key, "DataplaneClusterName")
	assert.Equal(t, cfg.ManagedDB.Tags[0].Value, "acs-dev-dp-01")
}
