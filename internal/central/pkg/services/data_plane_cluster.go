package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// DataPlaneClusterService ...
type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(clusterID string, status dbapi.DataPlaneClusterStatus) *errors.ServiceError
	GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError)
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

type dataPlaneClusterService struct {
	di.Inject
	ClusterService         ClusterService
	CentralConfig          *config.CentralConfig
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

	return &dbapi.DataPlaneClusterConfig{}, nil
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
