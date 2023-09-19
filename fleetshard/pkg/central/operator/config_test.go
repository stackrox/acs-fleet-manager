package operator

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"testing"
)

func getExampleConfig() []byte {
	return []byte(`
crd:
  baseURL: https://raw.githubusercontent.com/stackrox/stackrox/{{ .GitRef }}/operator/bundle/manifests/
  gitRef: 4.1.1
operators:
- gitRef: 4.1.1
  image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
  helmValues: |
    operator:
      resources:
        requests:
          memory: 500Mi
          cpu: 50m
`)
}

func TestGetOperatorConfig(t *testing.T) {
	conf, err := parseConfig(getExampleConfig())
	require.NoError(t, err)
	assert.Len(t, conf.Configs, 1)
	assert.Equal(t, "4.1.1", conf.Configs[0].GitRef)
	assert.Equal(t, "quay.io/rhacs-eng/stackrox-operator:4.1.1", conf.Configs[0].Image)
}

func TestValidateShouldSucceed(t *testing.T) {
	conf, err := parseConfig(getExampleConfig())
	require.NoError(t, err)

	err = Validate(field.NewPath("rhacsOperator"), conf)
	require.Nil(t, err)
}

func TestGetOperatorConfigFailsValidation(t *testing.T) {
	testCases := map[string]struct {
		getConfig func(*testing.T, OperatorConfigs) OperatorConfigs
		contains  string
	}{
		"should fail with invalid baseURL not able to download CRD": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				config.CRD.BaseURL = "not an url"
				return config
			},
			contains: "failed downloading chart files",
		},
		"should fail with invalid git ref": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				config.Configs = []OperatorConfig{
					{GitRef: "%^-invalid", Image: "quay.io/rhacs-eng/test:4.0.0", HelmValues: ""},
				}
				return config
			},
			contains: "failed to parse images: label selector %^-invalid is not valid",
		},
		"should fail with invalid image": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				config.Configs = []OperatorConfig{
					{GitRef: "4.0.0", Image: "quay.io//invalid", HelmValues: ""},
				}
				return config
			},
			contains: "failed to parse images: invalid reference format",
		},
		"should fail with invalid helm values": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				config.Configs = []OperatorConfig{
					{GitRef: "4.0.0", Image: "quay.io/rhacs-eng/test:4.0.0", HelmValues: "invalid YAML"},
				}
				return config
			},
			contains: "Unmarshalling Helm values failed for operator 4.0.0",
		},
	}

	for key, testCase := range testCases {
		t.Run(key, func(t *testing.T) {
			config, err := parseConfig(getExampleConfig())
			require.NoError(t, err)

			err = Validate(field.NewPath("rhacsOperator"), testCase.getConfig(t, config))
			require.NotNil(t, err)
			require.NotEmpty(t, testCase.contains)
			assert.Contains(t, err.Error(), testCase.contains)
		})
	}
}
