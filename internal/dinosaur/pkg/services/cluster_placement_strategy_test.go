package services

import (
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
			expectedType:    &FirstReadyPlacementStrategy{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.description, func(t *testing.T) {
			strategy := NewClusterPlacementStrategy(tc.createClusterService())

			require.IsType(t, tc.expectedType, strategy)
		})
	}
}

func TestFirstClusterPlacementStrategy(t *testing.T) {
	centralRequest := buildCentralRequest(func(centralRequest *dbapi.CentralRequest) {
		centralRequest.InstanceType = "standard"
	})

	notSchedulable := buildCluster(func(cluster *api.Cluster) {
		cluster.ClusterID = "notSchedulable"
		cluster.SupportedInstanceType = "standard,eval"
		cluster.Schedulable = false
	})
	notSupported := buildCluster(func(cluster *api.Cluster) {
		cluster.ClusterID = "notSupported"
		cluster.SupportedInstanceType = "eval"
		cluster.Schedulable = true
	})
	goodCluster1 := buildCluster(func(cluster *api.Cluster) {
		cluster.ClusterID = "good1"
		cluster.SupportedInstanceType = "standard,eval"
		cluster.Schedulable = true
	})
	goodCluster2 := buildCluster(func(cluster *api.Cluster) {
		cluster.ClusterID = "good2"
		cluster.SupportedInstanceType = "standard,eval"
		cluster.Schedulable = true
	})

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
			central:         centralRequest,
			expectedError:   serviceErrors.New(apiErrors.ErrorGeneral, "error in FindAllClusters"),
			expectedCluster: nil,
		},
		{
			description: "should return nil if clusters is empty",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{}, nil
					},
				}
			},
			central:         centralRequest,
			expectedError:   nil,
			expectedCluster: nil,
		},
		{
			description: "should return nil if no cluster supporting central instancetype was found",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{notSupported}, nil
					},
				}
			},
			central:         centralRequest,
			expectedError:   nil,
			expectedCluster: nil,
		},
		{
			description: "should return nil if no cluster is schedulable",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{notSchedulable}, nil
					},
				}
			},
			central:         centralRequest,
			expectedError:   nil,
			expectedCluster: nil,
		},
		{
			description: "should return first good cluster",
			newClusterServiceMock: func() ClusterService {
				return &ClusterServiceMock{
					FindAllClustersFunc: func(criteria FindClusterCriteria) ([]*api.Cluster, *serviceErrors.ServiceError) {
						return []*api.Cluster{notSupported, goodCluster1, notSchedulable, goodCluster2}, nil
					},
				}
			},
			central:         centralRequest,
			expectedError:   nil,
			expectedCluster: goodCluster1,
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
