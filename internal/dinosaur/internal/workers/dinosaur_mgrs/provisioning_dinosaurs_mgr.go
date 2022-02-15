package dinosaur_mgrs

import (
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/google/uuid"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	"github.com/pkg/errors"

	"github.com/golang/glog"
)

// ProvisioningDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ProvisioningDinosaurManager struct {
	workers.BaseWorker
	dinosaurService      services.DinosaurService
	observatoriumService services.ObservatoriumService
}

// NewProvisioningDinosaurManager creates a new dinosaur manager
func NewProvisioningDinosaurManager(dinosaurService services.DinosaurService, observatoriumService services.ObservatoriumService) *ProvisioningDinosaurManager {
	return &ProvisioningDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "provisioning_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:      dinosaurService,
		observatoriumService: observatoriumService,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *ProvisioningDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *ProvisioningDinosaurManager) Stop() {
	k.StopWorker(k)
}

func (k *ProvisioningDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling dinosaurs")
	var encounteredErrors []error

	// handle provisioning dinosaurs state
	// Dinosaurs in a "provisioning" state means that it is ready to be sent to the Fleetshard Operator for Dinosaur creation in the data plane cluster.
	// The update of the Dinosaur request status from 'provisioning' to another state will be handled by the Fleetshard Operator.
	// We only need to update the metrics here.
	provisioningDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusProvisioning)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list provisioning dinosaurs"))
	} else {
		glog.Infof("provisioning dinosaurs count = %d", len(provisioningDinosaurs))
	}
	for _, dinosaur := range provisioningDinosaurs {
		glog.V(10).Infof("provisioning dinosaur id = %s", dinosaur.ID)
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusProvisioning, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		// TODO implement additional reconcilation logic for provisioning dinosaurs
	}

	return encounteredErrors
}
