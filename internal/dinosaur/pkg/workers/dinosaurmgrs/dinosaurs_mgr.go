package dinosaurmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	serviceErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// we do not add "deleted" status to the list as the dinosaurs are soft deleted once the status is set to "deleted", so no need to count them here.
var dinosaurMetricsStatuses = []constants2.DinosaurStatus{
	constants2.DinosaurRequestStatusAccepted,
	constants2.DinosaurRequestStatusPreparing,
	constants2.DinosaurRequestStatusProvisioning,
	constants2.DinosaurRequestStatusReady,
	constants2.DinosaurRequestStatusDeprovision,
	constants2.DinosaurRequestStatusDeleting,
	constants2.DinosaurRequestStatusFailed,
}

// DinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type DinosaurManager struct {
	workers.BaseWorker
	dinosaurService         services.DinosaurService
	accessControlListConfig *acl.AccessControlListConfig
	dinosaurConfig          *config.DinosaurConfig
}

// NewDinosaurManager creates a new dinosaur manager
func NewDinosaurManager(dinosaurService services.DinosaurService, accessControlList *acl.AccessControlListConfig, dinosaur *config.DinosaurConfig) *DinosaurManager {
	return &DinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "general_dinosaur_worker",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:         dinosaurService,
		accessControlListConfig: accessControlList,
		dinosaurConfig:          dinosaur,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *DinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *DinosaurManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *DinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling dinosaurs")
	var encounteredErrors []error

	// record the metrics at the beginning of the reconcile loop as some of the states like "accepted"
	// will likely gone after one loop. Record them at the beginning should give us more accurate metrics
	statusErrors := k.setDinosaurStatusCountMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	statusErrors = k.setClusterStatusCapacityUsedMetric()
	if len(statusErrors) > 0 {
		encounteredErrors = append(encounteredErrors, statusErrors...)
	}

	// delete dinosaurs of denied owners
	accessControlListConfig := k.accessControlListConfig
	if accessControlListConfig.EnableDenyList {
		glog.Infoln("reconciling denied dinosaur owners")
		dinosaurDeprovisioningForDeniedOwnersErr := k.reconcileDeniedDinosaurOwners(accessControlListConfig.DenyList)
		if dinosaurDeprovisioningForDeniedOwnersErr != nil {
			wrappedError := errors.Wrapf(dinosaurDeprovisioningForDeniedOwnersErr, "Failed to deprovision dinosaur for denied owners %s", accessControlListConfig.DenyList)
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	// cleaning up expired dinosaurs
	dinosaurConfig := k.dinosaurConfig
	if dinosaurConfig.DinosaurLifespan.EnableDeletionOfExpiredDinosaur {
		glog.Infoln("deprovisioning expired dinosaurs")
		expiredDinosaursError := k.dinosaurService.DeprovisionExpiredDinosaurs(dinosaurConfig.DinosaurLifespan.DinosaurLifespanInHours)
		if expiredDinosaursError != nil {
			wrappedError := errors.Wrap(expiredDinosaursError, "failed to deprovision expired Dinosaur instances")
			encounteredErrors = append(encounteredErrors, wrappedError)
		}
	}

	return encounteredErrors
}

func (k *DinosaurManager) reconcileDeniedDinosaurOwners(deniedUsers acl.DeniedUsers) *serviceErr.ServiceError {
	if len(deniedUsers) < 1 {
		return nil
	}

	return k.dinosaurService.DeprovisionDinosaurForUsers(deniedUsers)
}

func (k *DinosaurManager) setDinosaurStatusCountMetric() []error {
	counters, err := k.dinosaurService.CountByStatus(dinosaurMetricsStatuses)
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Dinosaurs by status")}
	}

	for _, c := range counters {
		metrics.UpdateDinosaurRequestsStatusCountMetric(c.Status, c.Count)
	}

	return nil
}

func (k *DinosaurManager) setClusterStatusCapacityUsedMetric() []error {
	regions, err := k.dinosaurService.CountByRegionAndInstanceType()
	if err != nil {
		return []error{errors.Wrap(err, "failed to count Dinosaurs by region")}
	}

	for _, region := range regions {
		used := float64(region.Count)
		metrics.UpdateClusterStatusCapacityUsedCount(region.Region, region.InstanceType, region.ClusterID, used)
	}

	return nil
}
