// Package dinosaurmgrs ...
package dinosaurmgrs

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

const acceptedCentralWorkerType = "accepted_dinosaur"

// AcceptedCentralManager represents a manager that periodically reconciles central requests
type AcceptedCentralManager struct {
	workers.BaseWorker
	centralService         services.DinosaurService
	quotaServiceFactory    services.QuotaServiceFactory
	clusterPlmtStrategy    services.ClusterPlacementStrategy
	dataPlaneClusterConfig *config.DataplaneClusterConfig
	centralRequestTimeout  time.Duration
}

// NewAcceptedCentralManager creates a new manager
func NewAcceptedCentralManager(centralService services.DinosaurService, quotaServiceFactory services.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, dataPlaneClusterConfig *config.DataplaneClusterConfig, centralRequestConfig *config.CentralRequestConfig) *AcceptedCentralManager {
	metrics.InitReconcilerMetricsForType(acceptedCentralWorkerType)
	return &AcceptedCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: acceptedCentralWorkerType,
			Reconciler: workers.Reconciler{},
		},
		centralService:         centralService,
		quotaServiceFactory:    quotaServiceFactory,
		clusterPlmtStrategy:    clusterPlmtStrategy,
		dataPlaneClusterConfig: dataPlaneClusterConfig,
		centralRequestTimeout:  centralRequestConfig.ExpirationTimeout,
	}
}

var acceptedCentralCount int32

// Reconcile ...
func (k *AcceptedCentralManager) Reconcile() []error {
	var encounteredErrors []error

	// handle accepted central requests
	acceptedCentralRequests, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted centrals"))
	}

	acceptedCentralCount = int32(len(acceptedCentralRequests))
	logger.InfoChangedInt32(&acceptedCentralCount, "accepted centrals count = %d", acceptedCentralCount)

	for _, centralRequest := range acceptedCentralRequests {
		glog.V(10).Infof("accepted central id = %s", centralRequest.ID)
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusAccepted, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		if err := k.reconcileAcceptedCentral(centralRequest); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted central %s", centralRequest.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedCentralManager) reconcileAcceptedCentral(centralRequest *dbapi.CentralRequest) error {
	// Check if instance creation is not expired before trying to reconcile it.
	// Otherwise, assign status Failed.
	if err := FailIfTimeoutExceeded(k.centralService, k.centralRequestTimeout, centralRequest); err != nil {
		return err
	}
	cluster, err := k.clusterPlmtStrategy.FindCluster(centralRequest)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster for central request %s", centralRequest.ID)
	}

	if cluster == nil {
		logger.Logger.Warningf("No available cluster found for Central instance with id %s", centralRequest.ID)
		return nil
	}

	centralRequest.ClusterID = cluster.ClusterID
	glog.Infof("Central instance with id %s is assigned to cluster with id %s", centralRequest.ID, centralRequest.ClusterID)

	if err := k.centralService.AcceptCentralRequest(centralRequest); err != nil {
		return errors.Wrapf(err, "failed to accept Central %s with cluster details", centralRequest.ID)
	}
	return nil
}
