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
rolloutGroups:
  - instanceIds:
    - id1
    - id2
    - id3
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
			name: "missing rollout group operator version",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("rolloutGroups[0]", "operatorVersion"), "operator version is required"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
rolloutGroups:
  - instanceIds:
    - id1
    - id2
`,
		}, {
			name: "empty rollout group instance ids (empty string)",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("rolloutGroups[0]", "instanceIds[0]"), "instance ID is required"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
rolloutGroups:
  - instanceIds:
    - ""
    operatorVersion: v1.0.0
`,
		}, {
			name: "empty rollout group operator version (empty string)",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("rolloutGroups[0]", "operatorVersion"), "operator version is required"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
rolloutGroups:
  - instanceIds:
    - id1
    - id2
    operatorVersion: ""
`,
		}, {
			name: "duplicate instance id in rollout group",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Duplicate(field.NewPath("rolloutGroups").Index(0).Child("instanceIds").Index(2), "id2"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
rolloutGroups:
  - instanceIds:
    - id1
    - id2
    - id2
    operatorVersion: v1.0.0
`,
		}, {
			name: "duplicate instance ids across rollout groups",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Duplicate(field.NewPath("rolloutGroups").Index(1).Child("instanceIds").Index(0), "id1"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
rolloutGroups:
  - instanceIds:
    - id1
    - id2
    operatorVersion: v1.0.0
  - instanceIds:
    - id1
    - id3
    operatorVersion: v1.0.0
`,
		}, {
			name: "invalid yaml in patch",
			assert: func(t *testing.T, c *GitOpsConfig, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("overrides").Index(0).Child("patch"), "foo", "invalid patch"), err[0])
			},
			yaml: `
default:
  central: {}
  operatorVersion: v1.0.0
overrides:
  - instanceId: id1
    patch: foo
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

type validationMatch struct {
	path      *field.Path
	errorType field.ErrorType
}

func (v validationMatch) matches(err *field.Error) bool {
	return err.Type == v.errorType && err.Field == v.path.String()
}

func duplicate(path ...string) validationMatch {
	return validationMatch{path: field.NewPath(path[0], path[1:]...), errorType: field.ErrorTypeDuplicate}
}
func invalid(path ...string) validationMatch {
	return validationMatch{path: field.NewPath(path[0], path[1:]...), errorType: field.ErrorTypeInvalid}
}
func required(path ...string) validationMatch {
	return validationMatch{path: field.NewPath(path[0], path[1:]...), errorType: field.ErrorTypeRequired}
}
