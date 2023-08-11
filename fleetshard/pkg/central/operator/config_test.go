package operator

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestValidate(t *testing.T) {
	conf, err := parseConfig(getExampleConfig())
	require.NoError(t, err)

	errors := Validate(conf)
	assert.Len(t, errors, 0)
}
