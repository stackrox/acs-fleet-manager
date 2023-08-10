package gitops

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
		assert func(t *testing.T, c *Config, err field.ErrorList)
	}

	tests := []tc{
		{
			name: "valid",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Empty(t, err)
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: |
      {}`,
		}, {
			name: "invalid yaml in patch",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "foo", "invalid patch: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `foo` into v1alpha1.Central"), err[0])
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: foo
`,
		}, {
			name: "patch contains un-mergeable fields",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "spec: 123\n", "invalid patch: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!int `123` into v1alpha1.CentralSpec"), err[0])
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: |
      spec: 123
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Config
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &g))
			err := ValidateConfig(g)
			tt.assert(t, &g, err)
		})
	}
}
