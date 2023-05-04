package charts

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantResourcesChart(t *testing.T) {
	c, err := GetChart("tenant-resources")
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestDownloadTemplate(t *testing.T) {
	crdURL := "https://raw.githubusercontent.com/stackrox/stackrox/master/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml"
	err := DownloadTemplate(crdURL, "rhacs-operator")
	require.NoError(t, err)
	_, err = os.Stat("data/rhacs-operator/templates/platform.stackrox.io_securedclusters.yaml")
	require.NoError(t, err)

	c, err := GetChart("rhacs-operator")
	require.NoError(t, err)
	assert.NotNil(t, c)
}
