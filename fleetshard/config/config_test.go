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

func TestGetConfig_TenantAppSource(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		revision string
		path     string
	}{
		{
			name:     "defaults",
			repoURL:  "https://github.com/stackrox/acscs-tenant-resources.git",
			revision: "HEAD",
			path:     ".",
		},
		{
			name:     "explicit overrides",
			repoURL:  "https://github.com/example/custom-tenant-resources.git",
			revision: "release-1",
			path:     "charts/tenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLUSTER_ID", "some-value")
			t.Setenv("TENANT_DEFAULT_ARGOCD_APP_SOURCE_REPO_URL", "")
			t.Setenv("TENANT_DEFAULT_ARGOCD_APP_SOURCE_TARGET_REVISION", "")
			t.Setenv("TENANT_DEFAULT_ARGOCD_APP_SOURCE_PATH", "")
			if tt.name == "explicit overrides" {
				t.Setenv("TENANT_DEFAULT_ARGOCD_APP_SOURCE_REPO_URL", tt.repoURL)
				t.Setenv("TENANT_DEFAULT_ARGOCD_APP_SOURCE_TARGET_REVISION", tt.revision)
				t.Setenv("TENANT_DEFAULT_ARGOCD_APP_SOURCE_PATH", tt.path)
			}

			cfg, err := GetConfig()
			require.NoError(t, err)
			assert.Equal(t, tt.repoURL, cfg.TenantDefaultArgoCdAppSourceRepoURL)
			assert.Equal(t, tt.revision, cfg.TenantDefaultArgoCdAppSourceTargetRevision)
			assert.Equal(t, tt.path, cfg.TenantDefaultArgoCdAppSourcePath)
		})
	}
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
	require.Len(t, cfg.ManagedDB.SharedTags, 1)
	assert.Equal(t, cfg.ManagedDB.SharedTags[0].Key, "DataplaneClusterName")
	assert.Equal(t, cfg.ManagedDB.SharedTags[0].Value, "acs-dev-dp-01")
}
