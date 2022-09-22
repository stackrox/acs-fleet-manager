package centralmgrs

import (
	"time"

	"github.com/google/uuid"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

// ProvisioningCentralManager represents a central manager that periodically reconciles central requests
type ProvisioningCentralManager struct {
	workers.BaseWorker
	centralService       services.CentralService
	observatoriumService services.ObservatoriumService
}

// NewProvisioningCentralManager creates a new central manager
func NewProvisioningCentralManager(centralService services.CentralService, observatoriumService services.ObservatoriumService) *ProvisioningCentralManager {
	return &ProvisioningCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "provisioning_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		centralService:       centralService,
		observatoriumService: observatoriumService,
	}
}

// Start initializes the central manager to reconcile central requests
func (k *ProvisioningCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *ProvisioningCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *ProvisioningCentralManager) Reconcile() []error {
	glog.Infoln("reconciling centrals")
	var encounteredErrors []error

	// handle provisioning centrals state
	// centrals in a "provisioning" state means that it is ready to be sent to the Fleetshard Operator for central creation in the data plane cluster.
	// The update of the central request status from 'provisioning' to another state will be handled by the Fleetshard Operator.
	// We only need to update the metrics here.
	provisioningCentrals, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusProvisioning)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list provisioning centrals"))
	} else {
		glog.Infof("provisioning centrals count = %d", len(provisioningCentrals))
	}
	for _, central := range provisioningCentrals {
		glog.V(10).Infof("provisioning central id = %s", central.ID)
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusProvisioning, central.ID, central.ClusterID, time.Since(central.CreatedAt))
		// TODO implement additional reconcilation logic for provisioning centrals
	}

	return encounteredErrors
}
