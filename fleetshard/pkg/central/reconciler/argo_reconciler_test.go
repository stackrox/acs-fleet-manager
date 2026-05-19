package reconciler

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stretchr/testify/assert"
)

func TestSourceGetRepoURL(t *testing.T) {
	options := ArgoReconcilerOptions{TenantDefaultArgoCdAppSourceRepoURL: "default-repo-url"}
	r := argoReconciler{argoOpts: options}
	assert.Equal(t, "default-repo-url", r.getSourceRepoURL(private.ManagedCentral{}))
	assert.Equal(t, "custom-repo-url", r.getSourceRepoURL(private.ManagedCentral{
		Spec: private.ManagedCentralAllOfSpec{
			TenantResourcesValues: map[string]interface{}{
				"argoCd": map[string]interface{}{
					"sourceRepoUrl": "custom-repo-url",
				},
			},
		},
	}))
}

func TestGetSourcePath(t *testing.T) {
	options := ArgoReconcilerOptions{TenantDefaultArgoCdAppSourcePath: "default-source-path"}
	r := argoReconciler{argoOpts: options}
	assert.Equal(t, "default-source-path", r.getSourcePath(private.ManagedCentral{}))
	assert.Equal(t, "custom-source-path", r.getSourcePath(private.ManagedCentral{
		Spec: private.ManagedCentralAllOfSpec{
			TenantResourcesValues: map[string]interface{}{
				"argoCd": map[string]interface{}{
					"sourcePath": "custom-source-path",
				},
			},
		},
	}))
}

func TestGetSourceTargetRevision(t *testing.T) {
	options := ArgoReconcilerOptions{TenantDefaultArgoCdAppSourceTargetRevision: "default-revision"}
	r := argoReconciler{argoOpts: options}
	assert.Equal(t, "default-revision", r.getSourceTargetRevision(private.ManagedCentral{}))
	assert.Equal(t, "custom-revision", r.getSourceTargetRevision(private.ManagedCentral{
		Spec: private.ManagedCentralAllOfSpec{
			TenantResourcesValues: map[string]interface{}{
				"argoCd": map[string]interface{}{
					"sourceTargetRevision": "custom-revision",
				},
			},
		},
	}))
}

func TestDeclarativeConfigEnabled(t *testing.T) {
	rWithAuthProvider := argoReconciler{argoOpts: ArgoReconcilerOptions{WantsAuthProvider: true}}
	rWithoutAuthProvider := argoReconciler{argoOpts: ArgoReconcilerOptions{WantsAuthProvider: false}}

	t.Run("defaults to WantsAuthProvider for empty ManagedCentral", func(t *testing.T) {
		assert.True(t, rWithAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(private.ManagedCentral{}))
		assert.False(t, rWithoutAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(private.ManagedCentral{}))
	})

	t.Run("defaults to WantsAuthProvider for nil TenantResourcesValues", func(t *testing.T) {
		mc := private.ManagedCentral{
			Spec: private.ManagedCentralAllOfSpec{
				TenantResourcesValues: nil,
			},
		}
		assert.True(t, rWithAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
		assert.False(t, rWithoutAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
	})

	t.Run("defaults to WantsAuthProvider when declarativeConfig is missing", func(t *testing.T) {
		mc := private.ManagedCentral{
			Spec: private.ManagedCentralAllOfSpec{
				TenantResourcesValues: map[string]interface{}{
					"other": "value",
				},
			},
		}
		assert.True(t, rWithAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
		assert.False(t, rWithoutAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
	})

	t.Run("returns false when explicitly disabled", func(t *testing.T) {
		mc := private.ManagedCentral{
			Spec: private.ManagedCentralAllOfSpec{
				TenantResourcesValues: map[string]interface{}{
					"declarativeConfig": map[string]interface{}{
						"enabled": false,
					},
				},
			},
		}
		assert.False(t, rWithAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
		assert.False(t, rWithoutAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
	})

	t.Run("defaults to WantsAuthProvider when enabled is wrong type", func(t *testing.T) {
		mc := private.ManagedCentral{
			Spec: private.ManagedCentralAllOfSpec{
				TenantResourcesValues: map[string]interface{}{
					"declarativeConfig": map[string]interface{}{
						"enabled": "true",
					},
				},
			},
		}
		assert.True(t, rWithAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
		assert.False(t, rWithoutAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
	})

	t.Run("returns true when enabled", func(t *testing.T) {
		mc := private.ManagedCentral{
			Spec: private.ManagedCentralAllOfSpec{
				TenantResourcesValues: map[string]interface{}{
					"declarativeConfig": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		}
		assert.True(t, rWithAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
		assert.True(t, rWithoutAuthProvider.isArgoDeclarativeConfigReconciliationEnabled(mc))
	})
}

func TestGetHelmValueByPath(t *testing.T) {
	values := map[string]interface{}{
		"topLevel": "top-value",
		"declarativeConfig": map[string]interface{}{
			"enabled": true,
			"count":   float64(42),
			"name":    "my-config",
			"nested": map[string]interface{}{
				"deep":      "deep-value",
				"flag":      false,
				"threshold": 3.14,
			},
		},
		"wrongType": 123,
		"intSection": map[string]interface{}{
			"port":  8080,
			"count": 3,
		},
	}

	t.Run("string values", func(t *testing.T) {
		assert.Equal(t, "top-value", getHelmValueByPath(values, "topLevel", "default"))
		assert.Equal(t, "my-config", getHelmValueByPath(values, "declarativeConfig.name", "default"))
		assert.Equal(t, "deep-value", getHelmValueByPath(values, "declarativeConfig.nested.deep", "default"))
	})

	t.Run("bool values", func(t *testing.T) {
		assert.Equal(t, true, getHelmValueByPath(values, "declarativeConfig.enabled", false))
		assert.Equal(t, false, getHelmValueByPath(values, "declarativeConfig.nested.flag", true))
	})

	t.Run("int values", func(t *testing.T) {
		assert.Equal(t, 8080, getHelmValueByPath(values, "intSection.port", 0))
		assert.Equal(t, 3, getHelmValueByPath(values, "intSection.count", 0))
		assert.Equal(t, 0, getHelmValueByPath(values, "intSection.missing", 0))
		// float64 value doesn't match int
		assert.Equal(t, 0, getHelmValueByPath(values, "declarativeConfig.count", 0))
	})

	t.Run("float64 values", func(t *testing.T) {
		assert.Equal(t, float64(42), getHelmValueByPath(values, "declarativeConfig.count", float64(0)))
		assert.Equal(t, 3.14, getHelmValueByPath(values, "declarativeConfig.nested.threshold", float64(0)))
	})

	t.Run("returns default for missing key", func(t *testing.T) {
		assert.Equal(t, "default", getHelmValueByPath(values, "nonexistent", "default"))
		assert.Equal(t, "default", getHelmValueByPath(values, "declarativeConfig.missing", "default"))
		assert.Equal(t, "default", getHelmValueByPath(values, "declarativeConfig.nested.missing", "default"))
	})

	t.Run("returns default for type mismatch", func(t *testing.T) {
		assert.Equal(t, "default", getHelmValueByPath(values, "declarativeConfig.enabled", "default"))
		assert.Equal(t, false, getHelmValueByPath(values, "declarativeConfig.name", false))
	})

	t.Run("returns default when intermediate key is not a map", func(t *testing.T) {
		assert.Equal(t, "default", getHelmValueByPath(values, "topLevel.child", "default"))
		assert.Equal(t, "default", getHelmValueByPath(values, "wrongType.child", "default"))
	})

	t.Run("returns default for nil values map", func(t *testing.T) {
		assert.Equal(t, "default", getHelmValueByPath[string](nil, "any.path", "default"))
		assert.Equal(t, true, getHelmValueByPath[bool](nil, "any.path", true))
	})

	t.Run("returns default for empty path", func(t *testing.T) {
		assert.Equal(t, "default", getHelmValueByPath(values, "", "default"))
	})
}
