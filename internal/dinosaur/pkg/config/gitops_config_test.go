package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

func TestValidateGitOpsConfig(t *testing.T) {
	type tc struct {
		name   string
		yaml   string
		assert func(t *testing.T, c *GitOpsConfig, err field.ErrorList)
	}

	tests := []tc{
		{
			name: "valid",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Empty(t, err)
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
overrides:
  - instanceId: id1
    patch: |
      {}`,
		}, {
			name: "missing default operator version",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("default", "operatorVersion"), "operator version is required"), err[0])
			},
			yaml: `
default:
  central: {}`,
		}, {
			name: "invalid yaml in patch",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("overrides").Index(0).Child("patch"), "foo", "invalid patch: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `foo` into v1alpha1.Central"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
overrides:
  - instanceId: id1
    patch: foo
`,
		}, {
			name: "patch contains un-mergeable fields",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("overrides").Index(0).Child("patch"), "spec: 123\n", "invalid patch: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!int `123` into v1alpha1.CentralSpec"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
overrides:
  - instanceId: id1
    patch: |
      spec: 123
`,
		}, {
			name: "duplicate override instance ID",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Duplicate(field.NewPath("overrides").Index(1).Child("instanceId"), "id1"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
overrides:
  - instanceId: id1
    patch: |
       {}
  - instanceId: id1
    patch: |
      {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g GitOpsConfig
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &g))
			err := ValidateGitOpsConfig(g)
			tt.assert(t, &g, err)
		})
	}
}
