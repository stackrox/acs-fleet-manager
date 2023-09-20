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
rhacsOperators:
  crd:
    baseURL: https://raw.githubusercontent.com/stackrox/stackrox/{{ .GitRef }}/operator/bundle/manifests/
    gitRef: 4.1.1
  operators:
  - gitRef: 4.1.1
    image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
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
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "foo", "invalid patch: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type v1alpha1.Central"), err[0])
			},
			yaml: `
rhacsOperators:
  crd:
    baseURL: https://raw.githubusercontent.com/stackrox/stackrox/{{ .GitRef }}/operator/bundle/manifests/
    gitRef: 4.1.1
  operators:
  - gitRef: 4.1.1
    image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
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
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "spec: 123\n", "invalid patch: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal number into Go struct field Central.spec of type v1alpha1.CentralSpec"), err[0])
			},
			yaml: `
rhacsOperators:
  crd:
    baseURL: https://raw.githubusercontent.com/stackrox/stackrox/{{ .GitRef }}/operator/bundle/manifests/
    gitRef: 4.1.1
  operators:
  - gitRef: 4.1.1
    image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
centrals:
  overrides:
  - instanceIds:
    patch: |
      spec: 123
`,
		}, {
			name: "invalid operator config and central config",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Contains(t, err.ToAggregate().Errors()[0].Error(), "cannot unmarshal string into Go value of type v1alpha1.Central", "central config was not validated")
			},
			yaml: `
rhacsOperators:
  crd:
    baseURL: invalid
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: invalid
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
