package dinosaurmgrs

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	serviceErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// GracePeriodManager represents a dinosaur manager that manages grace period date.
type GracePeriodManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
}

// NewGracePeriodManager creates a new grace period manager
func NewGracePeriodManager(dinosaurService services.DinosaurService) *DinosaurManager {
	return &DinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "grace_period_worker",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService: dinosaurService,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *GracePeriodManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *GracePeriodManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *GracePeriodManager) Reconcile() []error {
	glog.Infoln("reconciling centrals")
	var encounteredErrors []error

	centrals, err := k.dinosaurService.ListByStatus(constants.GetDeletingStatuses()...)
	if err != nil {
		return append(encounteredErrors, err)
	}

	// reconciles grace_from field for central instances
	updateGraceFromErrors := k.reconcileCentralGraceFrom(centrals)
	if updateGraceFromErrors != nil {
		wrappedError := errors.Wrap(updateGraceFromErrors, "failed to update grace_from for central instances")
		encounteredErrors = append(encounteredErrors, wrappedError)
	}
	return encounteredErrors
}

// TODO: refactor arrays package.
func contains[T comparable](values []T, s T) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

func (k *GracePeriodManager) reconcileCentralGraceFrom(centrals dbapi.CentralList) serviceErr.ErrorList {
	glog.Infof("reconciling grace period start date for central instances")
	var svcErrors serviceErr.ErrorList
	subscriptionStatusByOrg := map[string]bool{}

	for _, central := range centrals {
		glog.Infof("reconciling grace_from for central instance %q", central.ID)

		// skip update when Central is marked for deletion or is already being deleted
		if contains(constants.GetDeletingStatuses(), constants.CentralStatus(central.Status)) {
			glog.Infof("central %q is in %q state, skipping grace_from reconciliation", central.ID, central.Status)
			continue
		}

		glog.Infof("checking quota entitlement status for Central instance %q", central.ID)
		active, exists := subscriptionStatusByOrg[central.OrganisationID]
		if !exists {
			isActive, err := k.dinosaurService.IsQuotaEntitlementActive(central)
			if err != nil {
				svcErrors = append(svcErrors, errors.Wrapf(err, "failed to get quota entitlement status of central instance %q", central.ID))
				continue
			}
			subscriptionStatusByOrg[central.OrganisationID] = isActive
			active = isActive
		}

		if err := k.updateGraceFromBasedOnQuotaEntitlement(central, active); err != nil {
			svcErrors = append(svcErrors, errors.Wrapf(err, "failed to update grace_from value based on quota entitlement for central instance %q", central.ID))
		}
	}

	return svcErrors
}

// Updates grace_from field of the given Central instance based on the user/organisation's quota entitlement status
func (k *GracePeriodManager) updateGraceFromBasedOnQuotaEntitlement(central *dbapi.CentralRequest, isQuotaEntitlementActive bool) error {
	// if quota entitlement is active, ensure grace_from is set to null
	if isQuotaEntitlementActive && central.GraceFrom != nil {
		glog.Infof("updating grace start date of central instance %q to NULL", central.ID)
		return k.dinosaurService.Updates(central, map[string]interface{}{
			"grace_from": nil,
		})
	}

	// if quota entitlement is not active and grace_from is not already set, set its value based on the current time and grace period allowance
	if !isQuotaEntitlementActive && central.GraceFrom == nil {
		graceFromTime := time.Now()
		glog.Infof("quota entitlement for central instance %q is no longer active, updating grace_from to %q", central.ID, graceFromTime.Format(time.RFC1123Z))
		return k.dinosaurService.Updates(central, map[string]interface{}{
			"grace_from": &graceFromTime,
		})
	}

	glog.Infof("no grace_from changes needed for central %q, skipping update", central.ID)
	return nil
}
