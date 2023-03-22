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
			dataPlaneConfig: &config.DataplaneClusterConfig{},
			expectedType:    FirstReadyPlacementStrategy{},
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
			central:         buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {}),
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
			central:         buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {}),
			expectedError:   errors.New("no schedulable cluster found"),
			expectedCluster: nil,
		},
		{
			description: "should return error if no clusters with SkipScheduling false was found",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{
							buildCluster(func(cluster *api.Cluster) {
								cluster.SkipScheduling = true
							}),
							buildCluster(func(cluster *api.Cluster) {
								cluster.SkipScheduling = true
							}),
						}, nil
					},
				}
			},
			central:         buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {}),
			expectedError:   errors.New("no schedulable cluster found"),
			expectedCluster: nil,
		},
		{
			description: "should return error if no cluster supporting central instancetype was found",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{
							buildCluster(func(cluster *api.Cluster) {}),
							buildCluster(func(cluster *api.Cluster) {}),
						}, nil
					},
				}
			},
			central: buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {
				centralRequest.InstanceType = "standard"
			}),
			expectedError:   errors.New("no schedulable cluster found"),
			expectedCluster: nil,
		},
		{
			description: "should return first schedulable cluster",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{
							buildCluster(func(cluster *api.Cluster) {}),
							buildCluster(func(cluster *api.Cluster) {
								cluster.SupportedInstanceType = "standard,eval"
							}),
						}, nil
					},
				}
			},
			central: buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {
				centralRequest.InstanceType = "standard"
			}),
			expectedError: nil,
			expectedCluster: buildCluster(func(cluster *api.Cluster) {
				cluster.SupportedInstanceType = "standard,eval"
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.description, func(t *testing.T) {
			strategy := FirstReadyPlacementStrategy{clusterService: tc.newClusterServiceMock()}
			cluster, err := strategy.FindCluster(tc.central)
			require.Equal(t, err, tc.expectedError)
			if tc.expectedError != nil {
				require.Nil(t, cluster)
			}

			if cluster != nil {
				require.Equal(t, *tc.expectedCluster, *cluster)
			}

		})

	}
}
