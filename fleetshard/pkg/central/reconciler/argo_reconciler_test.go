package reconciler

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"testing"
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
