package services

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"

	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError
	GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError)
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

const dataPlaneClusterStatusCondReadyName = "Ready"

type dataPlaneClusterService struct {
	di.Inject
	ClusterService         ClusterService
	DinosaurConfig         *config.DinosaurConfig
	ObservabilityConfig    *observatorium.ObservabilityConfiguration
	DataplaneClusterConfig *config.DataplaneClusterConfig
}

func NewDataPlaneClusterService(config dataPlaneClusterService) *dataPlaneClusterService {
	return &config
}

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

func (d *dataPlaneClusterService) UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError {
	cluster, svcErr := d.ClusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("Cluster agent with ID '%s' not found", clusterID)
	}

	// TODO(create-ticket): restore when cluster transition to ready is implemented
	/*
	if !d.clusterCanProcessStatusReports(cluster) {
		glog.V(10).Infof("Cluster with ID '%s' is in '%s' state. Ignoring status report...", clusterID, cluster.Status)
		return nil
	}*/

	fleetShardOperatorReady, err := d.isFleetShardOperatorReady(status)
	if err != nil {
		return errors.ToServiceError(err)
	}
	if !fleetShardOperatorReady {
		if cluster.Status != api.ClusterWaitingForFleetShardOperator {
			err := d.ClusterService.UpdateStatus(*cluster, api.ClusterWaitingForFleetShardOperator)
			if err != nil {
				return errors.ToServiceError(err)
			}
			metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterWaitingForFleetShardOperator)
		}
		glog.V(10).Infof("Fleet Shard Operator not ready for Cluster ID '%s", clusterID)
		return nil
	}

	// We calculate the status based on the stats received by the Fleet operator
	// BEFORE performing the scaling actions. If scaling actions are performed later
	// then it will be reflected on the next data plane cluster status report
	err = d.setClusterStatus(cluster, status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	return nil
}

func (d *dataPlaneClusterService) setClusterStatus(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) error {
	prevAvailableDinosaurOperatorVersions, err := cluster.GetAvailableDinosaurOperatorVersions()
	if err != nil {
		return err
	}
	if len(status.AvailableDinosaurOperatorVersions) > 0 && !reflect.DeepEqual(prevAvailableDinosaurOperatorVersions, status.AvailableDinosaurOperatorVersions) {
		err := cluster.SetAvailableDinosaurOperatorVersions(status.AvailableDinosaurOperatorVersions)
		if err != nil {
			return err
		}
		glog.Infof("Updating Dinosaur operator available versions for cluster ID '%s'. Versions: '%v'\n", cluster.ClusterID, status.AvailableDinosaurOperatorVersions)
		svcErr := d.ClusterService.Update(*cluster)
		if svcErr != nil {
			return err
		}
	}

	if cluster.Status != api.ClusterReady {
		clusterIsWaitingForFleetShardOperator := cluster.Status == api.ClusterWaitingForFleetShardOperator
		err := d.ClusterService.UpdateStatus(*cluster, api.ClusterReady)
		if err != nil {
			return err
		}
		if clusterIsWaitingForFleetShardOperator {
			metrics.UpdateClusterCreationDurationMetric(metrics.JobTypeClusterCreate, time.Since(cluster.CreatedAt))
		}
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterReady)
	}

	return nil
}

func (d *dataPlaneClusterService) clusterCanProcessStatusReports(cluster *api.Cluster) bool {
	return cluster.Status == api.ClusterReady ||
		cluster.Status == api.ClusterComputeNodeScalingUp ||
		cluster.Status == api.ClusterFull ||
		cluster.Status == api.ClusterWaitingForFleetShardOperator
}

func (d *dataPlaneClusterService) isFleetShardOperatorReady(status *dbapi.DataPlaneClusterStatus) (bool, error) {
	for _, cond := range status.Conditions {
		if cond.Type == dataPlaneClusterStatusCondReadyName {
			condVal, err := strconv.ParseBool(cond.Status)
			if err != nil {
				return false, err
			}
			return condVal, nil
		}
	}
	return false, nil
}
