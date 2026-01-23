package centralmgrs

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"

	"github.com/google/uuid"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

const provisioningCentralWorkerType = "provisioning_central"

// ProvisioningCentralManager represents a central manager that periodically reconciles central requests
type ProvisioningCentralManager struct {
	workers.BaseWorker
	centralService        services.CentralService
	centralRequestTimeout time.Duration
}

// NewProvisioningCentralManager creates a new central manager
func NewProvisioningCentralManager(centralService services.CentralService, centralRequestConfig *config.CentralRequestConfig) *ProvisioningCentralManager {
	metrics.InitReconcilerMetricsForType(provisioningCentralWorkerType)
	return &ProvisioningCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: provisioningCentralWorkerType,
			Reconciler: workers.Reconciler{},
		},
		centralService:        centralService,
		centralRequestTimeout: centralRequestConfig.ExpirationTimeout,
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
	var encounteredErrors []error

	// handle provisioning centrals state
	// Centrals in a "provisioning" state means that it is ready to be sent to the Fleetshard Sync for Central creation in the data plane cluster.
	// The update of the Central request status from 'provisioning' to another state will be handled by the Fleetshard Sync.
	// We only need to update the metrics here.
	provisioningCentrals, serviceErr := k.centralService.ListByStatus(constants.CentralRequestStatusProvisioning)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list provisioning centrals"))
	}
	if len(provisioningCentrals) > 0 {
		glog.Infof("provisioning centrals count = %d", len(provisioningCentrals))
	}
	for _, central := range provisioningCentrals {
		if err := FailIfTimeoutExceeded(k.centralService, k.centralRequestTimeout, central); err != nil {
			encounteredErrors = append(encounteredErrors, err)
		} else {
			glog.V(10).Infof("provisioning central id = %s", central.ID)
			metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants.CentralRequestStatusProvisioning, central.ID, central.ClusterID, time.Since(central.CreatedAt))
			// TODO implement additional reconcilation logic for provisioning centrals
		}
	}

	return encounteredErrors
}
