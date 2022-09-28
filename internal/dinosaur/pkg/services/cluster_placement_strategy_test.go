package services

import (
	"errors"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"

	apiErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stretchr/testify/require"

	serviceErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
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
			expectedType: FirstReadyPlacementStrategy{},
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

func TestFirstClusterPlacementStrategy(t *testing.T) {

	tt := []struct {
		description           string
		newClusterServiceMock func() ClusterService
		central               *dbapi.CentralRequest
		expectedError         error
		expectedCluster       *api.Cluster
	}{
		{
			description: "should return error if FindAllClusters returns error",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return nil, serviceErrors.New(apiErrors.ErrorGeneral, "error in FindAllClusters")
					},
				}
			},
			central:         buildDinosaurRequest(func(dinosaurRequest *dbapi.CentralRequest) {}),
			expectedError:   serviceErrors.New(apiErrors.ErrorGeneral, "error in FindAllClusters"),
			expectedCluster: nil,
		},
		{
			description: "should return error if clusters is empty",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{}, nil
					},
				}
			},
			central:         buildDinosaurRequest(func(dinosaurRequest *dbapi.CentralRequest) {}),
			expectedError:   errors.New("no schedulable cluster found"),
			expectedCluster: nil,
		},
		// should return error if no clusters with SkipScheduling true was found
		// should return error if no cluster supporting central instancetype was found
		// should return first ready cluster
	}

	for _, tc := range tt {
		t.Run(tc.description, func(t *testing.T) {
			strategy := FirstReadyPlacementStrategy{clusterService: tc.newClusterServiceMock()}
			cluster, err := strategy.FindCluster(tc.central)
			require.Equal(t, err, tc.expectedError)
			require.Equal(t, tc.expectedCluster, cluster)
		})

	}
}
