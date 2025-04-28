package centralmgrs

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	serviceErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// we do not add "deleted" status to the list as the centrals are soft deleted once the status is set to "deleted", so no need to count them here.
var centralMetricsStatuses = []constants.CentralStatus{
	constants.CentralRequestStatusAccepted,
	constants.CentralRequestStatusPreparing,
	constants.CentralRequestStatusProvisioning,
	constants.CentralRequestStatusReady,
	constants.CentralRequestStatusDeprovision,
	constants.CentralRequestStatusDeleting,
	constants.CentralRequestStatusFailed,
}

// CentralManager represents a central manager that periodically reconciles central requests
type CentralManager struct {
	workers.BaseWorker
	centralService          services.CentralService
	accessControlListConfig *acl.AccessControlListConfig
	centralConfig           *config.CentralConfig
}

// NewCentralManager creates a new central manager
func NewCentralManager(centralService services.CentralService, accessControlList *acl.AccessControlListConfig, centralConfig *config.CentralConfig) *CentralManager {
	return &CentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "general_central_worker",
			Reconciler: workers.Reconciler{},
		},
		centralService:          centralService,
		accessControlListConfig: accessControlList,
		centralConfig:           centralConfig,
	}
}

// Start initializes the central manager to reconcile central requests
func (k *CentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *CentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *CentralManager) Reconcile() []error {
	var encounteredErrors []error

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErrors := k.setCentralStatusCountMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	statusErrors = k.setClusterStatusCapacityUsedMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	// delete centrals of denied owners
	accessControlListConfig := k.accessControlListConfig
	if accessControlListConfig.EnableDenyList {
		centralDeprovisioningForDeniedOwnersErr := k.reconcileDeniedCentralOwners(accessControlListConfig.DenyList)
		if centralDeprovisioningForDeniedOwnersErr != nil {
			wrappedError := errors.Wrapf(centralDeprovisioningForDeniedOwnersErr, "Failed to deprovision central for denied owners %s", accessControlListConfig.DenyList)
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	// cleaning up expired centrals
	if svcErr := k.centralService.DeprovisionExpiredCentrals(); svcErr != nil {
		encounteredErrors = append(encounteredErrors,
			errors.Wrap(svcErr, "failed to deprovision expired Central instances"))
	}

	return encounteredErrors
}

func (k *CentralManager) reconcileDeniedCentralOwners(deniedUsers acl.DeniedUsers) *serviceErr.ServiceError {
	if len(deniedUsers) < 1 {
		return nil
	}

	return k.centralService.DeprovisionCentralForUsers(deniedUsers)
}

func (k *CentralManager) setCentralStatusCountMetric() []error {
	counters, err := k.centralService.CountByStatus(centralMetricsStatuses)
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Centrals by status")}
	}

	for _, c := range counters {
		metrics.UpdateCentralRequestsStatusCountMetric(c.Status, c.Count)
	}

	return nil
}

func (k *CentralManager) setClusterStatusCapacityUsedMetric() []error {
	regions, err := k.centralService.CountByRegionAndInstanceType()
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Centrals by region")}
	}

	for _, region := range regions {
		used := float64(region.Count)
		metrics.UpdateClusterStatusCapacityUsedCount(region.Region, region.InstanceType, region.ClusterID, used)
	}

	return nil
}
