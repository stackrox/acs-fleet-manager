package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestGetOperatorConfigFailsValidation(t *testing.T) {
	testCases := map[string]struct {
		getConfig func(*testing.T, OperatorConfigs) OperatorConfigs
		contains  string
		success   bool
	}{
		"should fail with invalid crd url": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				config.CRDURLs = []string{
					"invalid url",
				}
				return config
			},
			contains: "failed downloading chart files",
		},
		"should fail with empty deployment name": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				cfg := validOperatorConfig()
				cfg[keyDeploymentName] = ""
				config.Configs = []OperatorConfig{cfg}
				return config
			},
			contains: "deployment name cannot be empty",
		},
		"should fail with invalid deployment name": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				cfg := validOperatorConfig()
				cfg[keyDeploymentName] = "!!"
				config.Configs = []OperatorConfig{cfg}
				return config
			},
			contains: "invalid deployment name",
		},
		"should fail with empty image": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				cfg := validOperatorConfig()
				cfg[keyImage] = ""
				config.Configs = []OperatorConfig{cfg}
				return config
			},
			contains: "image cannot be empty",
		},
		"should fail with invalid image": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				cfg := validOperatorConfig()
				cfg[keyImage] = "??"
				config.Configs = []OperatorConfig{cfg}
				return config
			},
			contains: "invalid image",
		},
		"validate should succeed with example config": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				return config
			},
			success: true,
		},
		"should succeed with empty operator configs": {
			getConfig: func(t *testing.T, config OperatorConfigs) OperatorConfigs {
				config.Configs = []OperatorConfig{}
				return config
			},
			success: true,
		},
	}

	for key, testCase := range testCases {
		t.Run(key, func(t *testing.T) {
			config, err := parseConfig(getExampleConfig())
			require.NoError(t, err)

			errList := Validate(field.NewPath("rhacsOperator"), testCase.getConfig(t, config))
			if testCase.contains != "" {
				require.Len(t, errList, 1)
				require.NotEmpty(t, testCase.contains)
				assert.Contains(t, errList.ToAggregate().Errors()[0].Error(), testCase.contains)
			} else {
				require.Nil(t, errList, "unexpected error: %v", errList.ToAggregate())
			}
		})
	}
}
