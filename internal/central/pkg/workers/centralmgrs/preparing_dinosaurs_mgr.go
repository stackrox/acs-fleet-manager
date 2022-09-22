package centralmgrs

import (
	"time"

	"github.com/google/uuid"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"

	serviceErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// PreparingCentralManager represents a central manager that periodically reconciles central requests
type PreparingCentralManager struct {
	workers.BaseWorker
	centralService services.CentralService
}

// NewPreparingCentralManager creates a new central manager
func NewPreparingCentralManager(centralService services.CentralService) *PreparingCentralManager {
	return &PreparingCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "preparing_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
	}
}

// Start initializes the central manager to reconcile central requests
func (k *PreparingCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *PreparingCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *PreparingCentralManager) Reconcile() []error {
	glog.Infoln("reconciling preparing centrals")
	var encounteredErrors []error

	// handle preparing centrals
	preparingCentrals, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusPreparing)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list preparing centrals"))
	} else {
		glog.Infof("preparing centrals count = %d", len(preparingCentrals))
	}

	for _, central := range preparingCentrals {
		glog.V(10).Infof("preparing central id = %s", central.ID)
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusPreparing, central.ID, central.ClusterID, time.Since(central.CreatedAt))
		if err := k.reconcilePreparingCentral(central); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile preparing central %s", central.ID))
			continue
		}

	}

	return encounteredErrors
}

func (k *PreparingCentralManager) reconcilePreparingCentral(central *dbapi.CentralRequest) error {
	if err := k.centralService.PrepareCentralRequest(central); err != nil {
		return k.handleCentralRequestCreationError(central, err)
	}

	return nil
}

func (k *PreparingCentralManager) handleCentralRequestCreationError(centralRequest *dbapi.CentralRequest, err *serviceErr.ServiceError) error {
	if err.IsServerErrorClass() {
		// retry the central creation request only if the failure is caused by server errors
		// and the time elapsed since its db record was created is still within the threshold.
		durationSinceCreation := time.Since(centralRequest.CreatedAt)
		if durationSinceCreation > constants2.CentralMaxDurationWithProvisioningErrs {
			metrics.IncreaseCentralTotalOperationsCountMetric(constants2.CentralOperationCreate)
			centralRequest.Status = string(constants2.CentralRequestStatusFailed)
			centralRequest.FailedReason = err.Reason
			updateErr := k.centralService.Update(centralRequest)
			if updateErr != nil {
				return errors.Wrapf(updateErr, "Failed to update central %s in failed state. Central failed reason %s", centralRequest.ID, centralRequest.FailedReason)
			}
			metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
			return errors.Wrapf(err, "Central %s is in server error failed state. Maximum attempts has been reached", centralRequest.ID)
		}
	} else if err.IsClientErrorClass() {
		metrics.IncreaseCentralTotalOperationsCountMetric(constants2.CentralOperationCreate)
		centralRequest.Status = string(constants2.CentralRequestStatusFailed)
		centralRequest.FailedReason = err.Reason
		updateErr := k.centralService.Update(centralRequest)
		if updateErr != nil {
			return errors.Wrapf(err, "Failed to update central %s in failed state", centralRequest.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		return errors.Wrapf(err, "error creating central %s", centralRequest.ID)
	}

	return errors.Wrapf(err, "failed to provision central %s on cluster %s", centralRequest.ID, centralRequest.ClusterID)
}
