package charts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantResourcesChart(t *testing.T) {
	c, err := GetChart("tenant-resources")
	require.NoError(t, err)
	assert.NotNil(t, c)
}
