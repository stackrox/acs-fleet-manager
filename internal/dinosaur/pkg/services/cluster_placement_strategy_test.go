package services

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestPlacementStrategyType(t *testing.T) {

	tt := []struct {
		description          string
		createClusterService func() ClusterService
		dataPlaneConfig      *config.DataplaneClusterConfig
		expectedType         interface{}
	}{
		{
			description: "DefaultClusterPlacementStrategy",
			createClusterService: func() ClusterService {
				return &ClusterServiceMock{}
			},
			dataPlaneConfig: &config.DataplaneClusterConfig{
				DataPlaneClusterTarget: "",
			},
			expectedType: DefaultClusterPlacementStrategy{},
		},
		{
			description: "TargetClusterPlacementStrategy",
			createClusterService: func() ClusterService {
				return &ClusterServiceMock{}
			},
			dataPlaneConfig: &config.DataplaneClusterConfig{
				DataPlaneClusterTarget: "test-cluster-id",
			},
			expectedType: TargetClusterPlacementStrategy{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.description, func(t *testing.T) {
			strategy := NewClusterPlacementStrategy(tc.createClusterService(), tc.dataPlaneConfig)

			require.IsType(t, tc.expectedType, strategy)
		})
	}
}
