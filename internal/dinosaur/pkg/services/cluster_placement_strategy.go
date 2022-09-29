package services

import (
	"errors"
	"fmt"
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
	var clusterSelection ClusterPlacementStrategy
	if dataplaneClusterConfig.DataPlaneClusterTarget != "" {
		clusterSelection = TargetClusterPlacementStrategy{
			targetClusterID: dataplaneClusterConfig.DataPlaneClusterTarget,
			clusterService:  clusterService}
	} else {
		clusterSelection = FirstReadyPlacementStrategy{
			clusterService: clusterService,
		}
	}

	return clusterSelection
}

// TODO(create-ticket): Revisit placement strategy before going live.
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
		if !c.SkipScheduling && supportsInstanceType(c, central.InstanceType) {
			return c, nil
		}
	}

	return nil, errors.New("no schedulable cluster found")
}

var _ ClusterPlacementStrategy = TargetClusterPlacementStrategy{}

// TargetClusterPlacementStrategy implements the ClusterPlacementStrategy to always return the same cluster
type TargetClusterPlacementStrategy struct {
	targetClusterID string
	clusterService  ClusterService
}

// FindCluster returns the target cluster of the placement strategy if found in the cluster list
func (f TargetClusterPlacementStrategy) FindCluster(central *dbapi.CentralRequest) (*api.Cluster, error) {
	cluster, err := f.clusterService.FindClusterByID(f.targetClusterID)
	if err != nil {
		return nil, err
	}

	if !supportsInstanceType(cluster, central.InstanceType) {
		return nil, fmt.Errorf("target cluster %s, does not support instance type %s", f.targetClusterID, central.InstanceType)
	}

	if cluster != nil {
		return cluster, nil
	}

	return nil, fmt.Errorf("target cluster %v not found in cluster list", f.targetClusterID)
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
