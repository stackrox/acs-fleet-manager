package dinosaur_mgrs

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

// AcceptedDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type AcceptedDinosaurManager struct {
	workers.BaseWorker
	dinosaurService        services.DinosaurService
	quotaServiceFactory    services.QuotaServiceFactory
	clusterPlmtStrategy    services.ClusterPlacementStrategy
	dataPlaneClusterConfig *config.DataplaneClusterConfig
}

// NewAcceptedDinosaurManager creates a new dinosaur manager
func NewAcceptedDinosaurManager(dinosaurService services.DinosaurService, quotaServiceFactory services.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, dataPlaneClusterConfig *config.DataplaneClusterConfig) *AcceptedDinosaurManager {
	return &AcceptedDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "accepted_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:        dinosaurService,
		quotaServiceFactory:    quotaServiceFactory,
		clusterPlmtStrategy:    clusterPlmtStrategy,
		dataPlaneClusterConfig: dataPlaneClusterConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *AcceptedDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *AcceptedDinosaurManager) Stop() {
	k.StopWorker(k)
}

func (k *AcceptedDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling accepted dinosaurs")
	var encounteredErrors []error

	// handle accepted dinosaurs
	acceptedDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted dinosaurs"))
	} else {
		glog.Infof("accepted dinosaurs count = %d", len(acceptedDinosaurs))
	}

	for _, dinosaur := range acceptedDinosaurs {
		glog.V(10).Infof("accepted dinosaur id = %s", dinosaur.ID)
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusAccepted, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		if err := k.reconcileAcceptedDinosaur(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted dinosaur %s", dinosaur.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedDinosaurManager) reconcileAcceptedDinosaur(dinosaur *dbapi.DinosaurRequest) error {
	glog.Infof("AcceptedDinosaurManager: %+v", k)
	glog.Infof("PLACEMENT: %+v", k.clusterPlmtStrategy)
	cluster, err := k.clusterPlmtStrategy.FindCluster(dinosaur)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster for dinosaur request %s", dinosaur.ID)
	}

	if cluster == nil {
		logger.Logger.Warningf("No available cluster found for Dinosaur instance with id %s", dinosaur.ID)
		return nil
	}

	dinosaur.ClusterID = cluster.ClusterID

	// Set desired dinosaur operator version
	var selectedDinosaurOperatorVersion *api.DinosaurOperatorVersion

	readyDinosaurOperatorVersions, err := cluster.GetAvailableAndReadyDinosaurOperatorVersions()
	if err != nil || len(readyDinosaurOperatorVersions) == 0 {
		// Dinosaur Operator version may not be available at the start (i.e. during upgrade of Dinosaur operator).
		// We need to allow the reconciler to retry getting and setting of the desired Dinosaur Operator version for a Dinosaur request
		// until the max retry duration is reached before updating its status to 'failed'.
		durationSinceCreation := time.Since(dinosaur.CreatedAt)
		if durationSinceCreation < constants2.AcceptedDinosaurMaxRetryDuration {
			glog.V(10).Infof("No available dinosaur operator version found for Dinosaur '%s' in Cluster ID '%s'", dinosaur.ID, dinosaur.ClusterID)
			return nil
		}
		dinosaur.Status = constants2.DinosaurRequestStatusFailed.String()
		if err != nil {
			err = errors.Wrapf(err, "failed to get desired dinosaur operator version %s", dinosaur.ID)
		} else {
			err = errors.New(fmt.Sprintf("failed to get desired dinosaur operator version %s", dinosaur.ID))
		}
		dinosaur.FailedReason = err.Error()
		if err2 := k.dinosaurService.Update(dinosaur); err2 != nil {
			return errors.Wrapf(err2, "failed to update failed dinosaur %s", dinosaur.ID)
		}
		return err
	}

	selectedDinosaurOperatorVersion = &readyDinosaurOperatorVersions[len(readyDinosaurOperatorVersions)-1]
	dinosaur.DesiredDinosaurOperatorVersion = selectedDinosaurOperatorVersion.Version

	// Set desired Dinosaur version
	if len(selectedDinosaurOperatorVersion.DinosaurVersions) == 0 {
		return errors.New(fmt.Sprintf("failed to get Dinosaur version %s", dinosaur.ID))
	}
	dinosaur.DesiredDinosaurVersion = selectedDinosaurOperatorVersion.DinosaurVersions[len(selectedDinosaurOperatorVersion.DinosaurVersions)-1].Version

	glog.Infof("Dinosaur instance with id %s is assigned to cluster with id %s", dinosaur.ID, dinosaur.ClusterID)
	dinosaur.Status = constants2.DinosaurRequestStatusPreparing.String()
	if err2 := k.dinosaurService.Update(dinosaur); err2 != nil {
		return errors.Wrapf(err2, "failed to update dinosaur %s with cluster details", dinosaur.ID)
	}
	return nil
}
