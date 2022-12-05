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
	assert.Equal(t, cfg.AuthType, "RHSSO")
	assert.Equal(t, cfg.RHSSORealm, "redhat-external")
	assert.Equal(t, cfg.RHSSOEndpoint, "https://sso.redhat.com")
	assert.Empty(t, cfg.OCMRefreshToken)
}

func TestSingleton_Failure(t *testing.T) {
	t.Cleanup(func() {
	})
	cfg, err := GetConfig()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestSingleton_Success_WhenManagedDBEnabled(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::012456789:role/fake_role")
	t.Setenv("MANAGED_DB_ENABLED", "true")
	t.Cleanup(func() {
		_ = os.Unsetenv("CLUSTER_ID")
		_ = os.Unsetenv("AWS_ROLE_ARN")
		_ = os.Unsetenv("MANAGED_DB_ENABLED")
	})
	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.AWSRoleARN, "arn:aws:iam::012456789:role/fake_role")
	assert.Equal(t, cfg.AWSRegion, "us-east-1")
}

func TestSingleton_Failure_WhenManagedDBEnabledAndAWSRoleArnNotSet(t *testing.T) {
	t.Setenv("CLUSTER_ID", "some-value")
	t.Setenv("MANAGED_DB_ENABLED", "true")
	t.Cleanup(func() {
		_ = os.Unsetenv("CLUSTER_ID")
		_ = os.Unsetenv("MANAGED_DB_ENABLED")
	})
	cfg, err := GetConfig()
	assert.Error(t, err, "MANAGED_DB_ENABLED == true and AWS_ROLE_ARN unset in the environment")
	assert.Nil(t, cfg)
}
