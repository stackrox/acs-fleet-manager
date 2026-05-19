package gitops

import (
	"testing"

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
