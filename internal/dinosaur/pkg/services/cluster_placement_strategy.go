package services

import (
	"errors"
	"strings"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

// ClusterPlacementStrategy ...
//
//go:generate moq -out cluster_placement_strategy_moq.go . ClusterPlacementStrategy
type ClusterPlacementStrategy interface {
	// FindCluster finds and returns a Cluster depends on the specific impl.
	FindCluster(central *dbapi.CentralRequest) (*api.Cluster, error)
}

// NewClusterPlacementStrategy return a concrete strategy impl. depends on the
// placement configuration. An appropriate ClusterPlacementStrategy implementation
// is returned based on the received parameters content
func NewClusterPlacementStrategy(clusterService ClusterService, dataplaneClusterConfig *config.DataplaneClusterConfig) ClusterPlacementStrategy {
	return &FirstReadyPlacementStrategy{clusterService: clusterService}
}

var _ ClusterPlacementStrategy = (*FirstReadyPlacementStrategy)(nil)

// FirstReadyPlacementStrategy ...
type FirstReadyPlacementStrategy struct {
	clusterService ClusterService
}

// FindCluster ...
func (d FirstReadyPlacementStrategy) FindCluster(central *dbapi.CentralRequest) (*api.Cluster, error) {
	clusters, err := d.clusterService.FindAllClusters(FindClusterCriteria{Status: api.ClusterReady})
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		if d.clusterService.IsClusterSchedulable(c.ID) && supportsInstanceType(c, central.InstanceType) {
			return c, nil
		}
	}

	return nil, errors.New("no schedulable cluster found")
}

func supportsInstanceType(c *api.Cluster, instanceType string) bool {
	supportedTypes := strings.Split(c.SupportedInstanceType, ",")
	for _, t := range supportedTypes {
		if t == instanceType {
			return true
		}
	}

	return false
}
