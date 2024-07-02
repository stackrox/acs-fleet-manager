package charts

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"testing"
)

func mustGetChart(t *testing.T, name string) *chart.Chart {
	t.Helper()
	chartFiles, err := TraverseChart(testdata, fmt.Sprintf("testdata/%s", name))
	require.NoError(t, err)
	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)
	return chart
}
