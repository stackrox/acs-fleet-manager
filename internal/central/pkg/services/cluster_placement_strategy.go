package services

import (
	"strings"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
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
func NewClusterPlacementStrategy(clusterService ClusterService) ClusterPlacementStrategy {
	return &FirstReadyPlacementStrategy{clusterService: clusterService}
}

var _ ClusterPlacementStrategy = (*FirstReadyPlacementStrategy)(nil)

// FirstReadyPlacementStrategy ...
type FirstReadyPlacementStrategy struct {
	clusterService ClusterService
}

// FindCluster ...
func (d FirstReadyPlacementStrategy) FindCluster(central *dbapi.CentralRequest) (*api.Cluster, error) {
	clusters, err := d.clusterService.FindAllClusters(centralToFindClusterCriteria(central))
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		if c.Schedulable && supportsInstanceType(c, central.InstanceType) {
			return c, nil
		}
	}

	return nil, nil
}

// AllMatchingClustersForCentral returns all cluster that fit the criteria to run a central
func AllMatchingClustersForCentral(central *dbapi.CentralRequest, clusterService ClusterService) ([]*api.Cluster, error) {
	clusters, err := clusterService.FindAllClusters(centralToFindClusterCriteria(central))
	if err != nil {
		return nil, err
	}

	matchingClusters := []*api.Cluster{}
	for _, c := range clusters {
		if c.Schedulable && supportsInstanceType(c, central.InstanceType) {
			matchingClusters = append(matchingClusters, c)
		}
	}

	return matchingClusters, nil
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

func centralToFindClusterCriteria(central *dbapi.CentralRequest) FindClusterCriteria {
	return FindClusterCriteria{
		Provider: central.CloudProvider,
		Region:   central.Region,
		MultiAZ:  central.MultiAZ,
		Status:   api.ClusterReady,
	}
}
