package operator

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func getExampleConfig() []byte {
	return []byte(`
crdUrls:
  - https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml
  - https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_centrals.yaml
operators:
- deploymentName: stackrox-operator
  image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
  labelSelector: "app.kubernetes.io/name=stackrox-operator"
  centralLabelSelector: "app.kubernetes.io/name=central"
  securedClusterLabelSelector: "app.kubernetes.io/name=securedCluster"
  disableCentralReconciler: true
  disableSecuredClusterReconciler: true
`)
}

func validOperatorConfig() OperatorConfig {
	return OperatorConfig{
		keyDeploymentName:         "stackrox-operator",
		keyImage:                  "quay.io/rhacs-eng/stackrox-operator:4.1.1",
		keyCentralLabelSelector:   "app.kubernetes.io/name=central",
		keySecuredClusterSelector: "app.kubernetes.io/name=securedCluster",
	}
}

func TestGetOperatorConfig(t *testing.T) {
	conf, err := parseConfig(getExampleConfig())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml",
		"https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_centrals.yaml",
	}, conf.CRDURLs)
	require.Len(t, conf.Configs, 1)
	operatorConfig := conf.Configs[0]
	assert.Equal(t, "stackrox-operator", operatorConfig.GetDeploymentName())
	assert.Equal(t, "quay.io/rhacs-eng/stackrox-operator:4.1.1", operatorConfig.GetImage())
	assert.Equal(t, "app.kubernetes.io/name=central", operatorConfig.GetCentralLabelSelector())
	assert.Equal(t, "app.kubernetes.io/name=securedCluster", operatorConfig.GetSecuredClusterLabelSelector())
	assert.True(t, operatorConfig.GetDisableCentralReconciler())
	assert.True(t, operatorConfig.GetDisableSecuredClusterReconciler())
}
