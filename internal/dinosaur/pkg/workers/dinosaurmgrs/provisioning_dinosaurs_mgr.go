package dinosaurmgrs

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"

	"github.com/google/uuid"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

const provisioningCentralWorkerType = "provisioning_dinosaur"

// ProvisioningDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ProvisioningDinosaurManager struct {
	workers.BaseWorker
	dinosaurService       services.DinosaurService
	observatoriumService  services.ObservatoriumService
	centralRequestTimeout time.Duration
}

// NewProvisioningDinosaurManager creates a new dinosaur manager
func NewProvisioningDinosaurManager(dinosaurService services.DinosaurService, observatoriumService services.ObservatoriumService, centralRequestConfig *config.CentralRequestConfig) *ProvisioningDinosaurManager {
	metrics.InitReconcilerMetricsForType(provisioningCentralWorkerType)
	return &ProvisioningDinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: provisioningCentralWorkerType,
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:       dinosaurService,
		observatoriumService:  observatoriumService,
		centralRequestTimeout: centralRequestConfig.ExpirationTimeout,
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

// Reconcile ...
func (k *ProvisioningDinosaurManager) Reconcile() []error {
	var encounteredErrors []error

	// handle provisioning dinosaurs state
	// Dinosaurs in a "provisioning" state means that it is ready to be sent to the Fleetshard Operator for Dinosaur creation in the data plane cluster.
	// The update of the Dinosaur request status from 'provisioning' to another state will be handled by the Fleetshard Operator.
	// We only need to update the metrics here.
	provisioningDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.CentralRequestStatusProvisioning)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list provisioning centrals"))
	}
	if len(provisioningDinosaurs) > 0 {
		glog.Infof("provisioning centrals count = %d", len(provisioningDinosaurs))
	}
	for _, dinosaur := range provisioningDinosaurs {
		if err := FailIfTimeoutExceeded(k.dinosaurService, k.centralRequestTimeout, dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, err)
		} else {
			glog.V(10).Infof("provisioning central id = %s", dinosaur.ID)
			metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusProvisioning, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
			// TODO implement additional reconcilation logic for provisioning dinosaurs
		}
	}

	return encounteredErrors
}
