package charts

import (
	"context"
	"embed"
	"testing"

	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
)

//go:embed testdata
var testdata embed.FS

func TestTenantResourcesChart(t *testing.T) {
	c, err := GetChart("tenant-resources")
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestInstallOrUpdateChart(t *testing.T) {
	chart, err := LoadChart(testdata, "testdata/test-chart")
	require.NoError(t, err)
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	ctx := context.Background()

	chartVals := chartutil.Values{
		"foo": "bar",
	}
	objs, err := RenderToObjects("test-release", "", chart, chartVals)
	require.NoError(t, err)
	obj := objs[0]

	err = InstallOrUpdateChart(ctx, obj, fakeClient)
	require.NoError(t, err)
}
