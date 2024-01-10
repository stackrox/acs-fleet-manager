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
  crdURls:
    - https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml
  operators:
    - image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
      deploymentName: "stackrox-operator"
      centralLabelSelector: "app.kubernetes.io/name=central"
      securedClusterLabelSelector: "app.kubernetes.io/name=securedCluster"
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
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "foo", "invalid patch: failed to apply override to Central instance: invalid JSON document"), err[0])
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
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "spec: 123\n", "invalid patch: failed to unmarshal Central instance: json: cannot unmarshal number into Go struct field Central.spec of type v1alpha1.CentralSpec"), err[0])
			},
			yaml: `
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
				assert.Contains(t, err.ToAggregate().Errors()[0].Error(), "invalid patch: failed to apply override to Central instance: invalid JSON document", "central config was not validated")
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: invalid
`,
		}, {
			name: "invalid memory request",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Contains(t, err.ToAggregate().Errors()[0].Error(), "invalid patch: failed to unmarshal Central instance: quantities must match the regular expression", "central config was not validated")
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: |
      spec:
        central:
          resources:
            requests:
              memory: "a"
`,
		},
		{
			name: "valid cluster config",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Empty(t, err)
			},
			yaml: `
dataPlaneClusters:
  - clusterID: 1234567890abcdef1234567890abcdef
    addons:
      - id: acs-fleetshard
`,
		},
		{
			name: "invalid cluster config when no clusterID",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("dataPlaneClusters").Index(0).Child("clusterID"), "clusterID is required"), err[0])
			},
			yaml: `
dataPlaneClusters:
  - addons:
      - id: acs-fleetshard
`,
		},
		{
			name: "invalid cluster config when no addon ID",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("dataPlaneClusters").Index(0).Child("addons").Index(0).Child("id"), "id is required"), err[0])
			},
			yaml: `
dataPlaneClusters:
  - clusterID: 1234567890abcdef1234567890abcdef
    addons:
      - version: 0.2.0
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
