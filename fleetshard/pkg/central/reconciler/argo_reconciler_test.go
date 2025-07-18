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

// TODO(ROX-30167): Remove this special case once all tenants are on ROSA
// This is to rename the central CR when creating the tenant on a fresh cluster.
// So that after the ROSA migration the CR name is not affected by the display name of the tenant
func TestOverrideCentralStaticCRName(t *testing.T) {
	tests := []struct {
		clusterName string
		values      map[string]interface{}
		expected    map[string]interface{}
	}{
		{
			clusterName: "acs-int-us-01",
			values:      map[string]interface{}{},
			expected:    map[string]interface{}{},
		},
		{
			clusterName: "acs-stage-dp-02",
			values:      map[string]interface{}{},
			expected:    map[string]interface{}{},
		},
		{
			clusterName: "acs-stage-eu-02",
			values:      map[string]interface{}{},
			expected:    map[string]interface{}{},
		},
		{
			clusterName: "acs-prod-dp-01",
			values:      map[string]interface{}{},
			expected:    map[string]interface{}{},
		},
		{
			clusterName: "acs-prod-eu-01",
			values:      map[string]interface{}{},
			expected:    map[string]interface{}{},
		},
		{
			clusterName: "acs-int-us-02",
			values:      map[string]interface{}{},
			expected: map[string]interface{}{
				"centralStaticCRName": true,
			},
		},
		{
			clusterName: "acs-int-us-02",
			values: map[string]interface{}{
				"centralStaticCRName": false,
			},
			expected: map[string]interface{}{
				"centralStaticCRName": false,
			},
		},
	}

	for _, tc := range tests {
		reconciler := argoReconciler{argoOpts: ArgoReconcilerOptions{
			ClusterName: tc.clusterName,
		}}

		val := tc.values
		reconciler.overrideCentralStaticCRName(val)
		assert.Equal(t, tc.expected, tc.values)
	}
}
