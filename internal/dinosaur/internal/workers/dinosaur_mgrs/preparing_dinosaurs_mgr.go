package dinosaur_mgrs

import (
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/golang/glog"

	serviceErr "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

// PreparingDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type PreparingDinosaurManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
}

// NewPreparingDinosaurManager creates a new dinosaur manager
func NewPreparingDinosaurManager(dinosaurService services.DinosaurService) *PreparingDinosaurManager {
	return &PreparingDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "preparing_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService: dinosaurService,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *PreparingDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *PreparingDinosaurManager) Stop() {
	k.StopWorker(k)
}

func (k *PreparingDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling preparing dinosaurs")
	var encounteredErrors []error

	// handle preparing dinosaurs
	preparingDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusPreparing)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list preparing dinosaurs"))
	} else {
		glog.Infof("preparing dinosaurs count = %d", len(preparingDinosaurs))
	}

	for _, dinosaur := range preparingDinosaurs {
		glog.V(10).Infof("preparing dinosaur id = %s", dinosaur.ID)
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusPreparing, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		if err := k.reconcilePreparingDinosaur(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile preparing dinosaur %s", dinosaur.ID))
			continue
		}

	}

	return encounteredErrors
}

func (k *PreparingDinosaurManager) reconcilePreparingDinosaur(dinosaur *dbapi.DinosaurRequest) error {
	if err := k.dinosaurService.PrepareDinosaurRequest(dinosaur); err != nil {
		return k.handleDinosaurRequestCreationError(dinosaur, err)
	}

	return nil
}

func (k *PreparingDinosaurManager) handleDinosaurRequestCreationError(dinosaurRequest *dbapi.DinosaurRequest, err *serviceErr.ServiceError) error {
	if err.IsServerErrorClass() {
		// retry the dinosaur creation request only if the failure is caused by server errors
		// and the time elapsed since its db record was created is still within the threshold.
		durationSinceCreation := time.Since(dinosaurRequest.CreatedAt)
		if durationSinceCreation > constants2.DinosaurMaxDurationWithProvisioningErrs {
			metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationCreate)
			dinosaurRequest.Status = string(constants2.DinosaurRequestStatusFailed)
			dinosaurRequest.FailedReason = err.Reason
			updateErr := k.dinosaurService.Update(dinosaurRequest)
			if updateErr != nil {
				return errors.Wrapf(updateErr, "Failed to update dinosaur %s in failed state. Dinosaur failed reason %s", dinosaurRequest.ID, dinosaurRequest.FailedReason)
			}
			metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusFailed, dinosaurRequest.ID, dinosaurRequest.ClusterID, time.Since(dinosaurRequest.CreatedAt))
			return errors.Wrapf(err, "Dinosaur %s is in server error failed state. Maximum attempts has been reached", dinosaurRequest.ID)
		}
	} else if err.IsClientErrorClass() {
		metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationCreate)
		dinosaurRequest.Status = string(constants2.DinosaurRequestStatusFailed)
		dinosaurRequest.FailedReason = err.Reason
		updateErr := k.dinosaurService.Update(dinosaurRequest)
		if updateErr != nil {
			return errors.Wrapf(err, "Failed to update dinosaur %s in failed state", dinosaurRequest.ID)
		}
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusFailed, dinosaurRequest.ID, dinosaurRequest.ClusterID, time.Since(dinosaurRequest.CreatedAt))
		return errors.Wrapf(err, "error creating dinosaur %s", dinosaurRequest.ID)
	}

	return errors.Wrapf(err, "failed to provision dinosaur %s on cluster %s", dinosaurRequest.ID, dinosaurRequest.ClusterID)
}
