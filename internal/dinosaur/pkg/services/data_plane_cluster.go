package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// DataPlaneClusterService ...
type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(clusterID string, status dbapi.DataPlaneClusterStatus) *errors.ServiceError
	GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError)
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

const dataPlaneClusterStatusCondReadyName = "Ready"

type dataPlaneClusterService struct {
	di.Inject
	ClusterService         ClusterService
	CentralConfig          *config.CentralConfig
	ObservabilityConfig    *observatorium.ObservabilityConfiguration
	DataplaneClusterConfig *config.DataplaneClusterConfig
}

// NewDataPlaneClusterService ...
func NewDataPlaneClusterService(config dataPlaneClusterService) DataPlaneClusterService {
	return &config
}

// GetDataPlaneClusterConfig ...
func (d *dataPlaneClusterService) GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError) {
	cluster, svcErr := d.ClusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return nil, svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return nil, errors.BadRequest("Cluster agent with ID '%s' not found", clusterID)
	}

	return &dbapi.DataPlaneClusterConfig{
		Observability: dbapi.DataPlaneClusterConfigObservability{
			AccessToken: d.ObservabilityConfig.ObservabilityConfigAccessToken,
			Channel:     d.ObservabilityConfig.ObservabilityConfigChannel,
			Repository:  d.ObservabilityConfig.ObservabilityConfigRepo,
			Tag:         d.ObservabilityConfig.ObservabilityConfigTag,
		},
	}, nil
}

// UpdateDataPlaneClusterStatus ...
func (d *dataPlaneClusterService) UpdateDataPlaneClusterStatus(clusterID string, status dbapi.DataPlaneClusterStatus) *errors.ServiceError {
	cluster, svcErr := d.ClusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("Cluster agent with ID '%s' not found", clusterID)
	}

	err := d.setClusterStatus(cluster, status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	return nil
}

func (d *dataPlaneClusterService) setClusterStatus(cluster *api.Cluster, status dbapi.DataPlaneClusterStatus) error {
	addonsJSON, err := json.Marshal(status.Addons)
	if err != nil {
		return fmt.Errorf("marshal fleetshardAddonStatus to JSON: %w", err)
	}
	cluster.Addons = addonsJSON
	if err := d.ClusterService.Update(*cluster); err != nil {
		return fmt.Errorf("updating cluster status: %w", err)
	}
	return nil
}

func (d *dataPlaneClusterService) clusterCanProcessStatusReports(cluster *api.Cluster) bool {
	return cluster.Status == api.ClusterReady ||
		cluster.Status == api.ClusterComputeNodeScalingUp ||
		cluster.Status == api.ClusterFull ||
		cluster.Status == api.ClusterWaitingForFleetShardOperator
}
