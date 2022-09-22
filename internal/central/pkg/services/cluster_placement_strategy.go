package services

import (
	"errors"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

// ClusterPlacementStrategy ...
//
//go:generate moq -out cluster_placement_strategy_moq.go . ClusterPlacementStrategy
type ClusterPlacementStrategy interface {
	// FindCluster finds and returns a Cluster depends on the specific impl.
	FindCluster(dinosaur *dbapi.CentralRequest) (*api.Cluster, error)
}

// NewClusterPlacementStrategy return a concrete strategy impl. depends on the
// placement configuration. An appropriate ClusterPlacementStrategy implementation
// is returned based on the received parameters content
func NewClusterPlacementStrategy(clusterService ClusterService, dataplaneClusterConfig *config.DataplaneClusterConfig) ClusterPlacementStrategy {
	var clusterSelection ClusterPlacementStrategy = FirstDBClusterPlacementStrategy{
		clusterService: clusterService,
	}

	return clusterSelection
}

// TODO(create-ticket): Revisit placement strategy before going live.
var _ ClusterPlacementStrategy = (*FirstDBClusterPlacementStrategy)(nil)

// FirstDBClusterPlacementStrategy ...
type FirstDBClusterPlacementStrategy struct {
	clusterService ClusterService
}

// FindCluster ...
func (d FirstDBClusterPlacementStrategy) FindCluster(dinosaur *dbapi.CentralRequest) (*api.Cluster, error) {
	clusters, err := d.clusterService.FindAllClusters(FindClusterCriteria{})
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, errors.New("no cluster was found")
	}
	return clusters[0], nil
}
