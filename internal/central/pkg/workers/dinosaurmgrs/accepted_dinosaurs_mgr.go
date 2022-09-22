package dinosaurmgrs

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

// AcceptedCentralManager represents a dinosaur manager that periodically reconciles dinosaur requests
type AcceptedCentralManager struct {
	workers.BaseWorker
	centralService         services.CentralService
	quotaServiceFactory    services.QuotaServiceFactory
	clusterPlmtStrategy    services.ClusterPlacementStrategy
	dataPlaneClusterConfig *config.DataplaneClusterConfig
}

// NewAcceptedCentralManager creates a new dinosaur manager
func NewAcceptedCentralManager(centralService services.CentralService, quotaServiceFactory services.QuotaServiceFactory, clusterPlmtStrategy services.ClusterPlacementStrategy, dataPlaneClusterConfig *config.DataplaneClusterConfig) *AcceptedCentralManager {
	return &AcceptedCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "accepted_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		centralService:         centralService,
		quotaServiceFactory:    quotaServiceFactory,
		clusterPlmtStrategy:    clusterPlmtStrategy,
		dataPlaneClusterConfig: dataPlaneClusterConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *AcceptedCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *AcceptedCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *AcceptedCentralManager) Reconcile() []error {
	glog.Infoln("reconciling accepted centrals")
	var encounteredErrors []error

	// handle accepted centrals
	acceptedCentrals, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusAccepted)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list accepted centrals"))
	} else {
		glog.Infof("accepted centrals count = %d", len(acceptedCentrals))
	}

	for _, central := range acceptedCentrals {
		glog.V(10).Infof("accepted central id = %s", central.ID)
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusAccepted, central.ID, central.ClusterID, time.Since(central.CreatedAt))
		if err := k.reconcileAcceptedCentral(central); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile accepted central %s", central.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *AcceptedCentralManager) reconcileAcceptedCentral(central *dbapi.CentralRequest) error {
	cluster, err := k.clusterPlmtStrategy.FindCluster(central)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster for central request %s", central.ID)
	}

	if cluster == nil {
		logger.Logger.Warningf("No available cluster found for Dinosaur instance with id %s", central.ID)
		return nil
	}

	central.ClusterID = cluster.ClusterID

	// Set desired central operator version
	var selectedDinosaurOperatorVersion *api.CentralOperatorVersion

	readyDinosaurOperatorVersions, err := cluster.GetAvailableAndReadyCentralOperatorVersions()
	if err != nil || len(readyDinosaurOperatorVersions) == 0 {
		// Dinosaur Operator version may not be available at the start (i.e. during upgrade of Dinosaur operator).
		// We need to allow the reconciler to retry getting and setting of the desired Dinosaur Operator version for a Dinosaur request
		// until the max retry duration is reached before updating its status to 'failed'.
		durationSinceCreation := time.Since(central.CreatedAt)
		if durationSinceCreation < constants2.AcceptedCentralMaxRetryDuration {
			glog.V(10).Infof("No available central operator version found for central '%s' in Cluster ID '%s'", central.ID, central.ClusterID)
			return nil
		}
		central.Status = constants2.CentralRequestStatusFailed.String()
		if err != nil {
			err = errors.Wrapf(err, "failed to get desired central operator version %s", central.ID)
		} else {
			err = errors.Errorf("failed to get desired central operator version %s", central.ID)
		}
		central.FailedReason = err.Error()
		if err2 := k.centralService.Update(central); err2 != nil {
			return errors.Wrapf(err2, "failed to update failed central %s", central.ID)
		}
		return err
	}

	selectedDinosaurOperatorVersion = &readyDinosaurOperatorVersions[len(readyDinosaurOperatorVersions)-1]
	central.DesiredCentralOperatorVersion = selectedDinosaurOperatorVersion.Version

	// Set desired Dinosaur version
	if len(selectedDinosaurOperatorVersion.CentralVersions) == 0 {
		return fmt.Errorf("failed to get Dinosaur version %s", central.ID)
	}
	central.DesiredCentralVersion = selectedDinosaurOperatorVersion.CentralVersions[len(selectedDinosaurOperatorVersion.CentralVersions)-1].Version

	glog.Infof("Central instance with id %s is assigned to cluster with id %s", central.ID, central.ClusterID)

	if err := k.centralService.AcceptCentralRequest(central); err != nil {
		return errors.Wrapf(err, "failed to accept Central %s with cluster details", central.ID)
	}
	return nil
}
