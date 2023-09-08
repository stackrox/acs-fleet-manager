package dinosaurmgrs

import (
	"database/sql"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	serviceErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// GracePeriodManager represents a dinosaur manager that manages grace period date.
type GracePeriodManager struct {
	workers.BaseWorker
	dinosaurService         services.DinosaurService
	accessControlListConfig *acl.AccessControlListConfig
	dinosaurConfig          *config.CentralConfig
}

// NewGracePeriodManager creates a new grace period manager
func NewGracePeriodManager(dinosaurService services.DinosaurService, accessControlList *acl.AccessControlListConfig, dinosaur *config.CentralConfig) *DinosaurManager {
	return &DinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "grace_period_worker",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:         dinosaurService,
		accessControlListConfig: accessControlList,
		dinosaurConfig:          dinosaur,
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

	// get all centrals and send their statuses to prometheus
	centrals, err := k.dinosaurService.ListAll()
	if err != nil {
		encounteredErrors = append(encounteredErrors, err)
	}

	// reconciles grace_from field for central instances
	updateGraceFromErrors := k.reconcileCentralGraceFrom(centrals)
	if updateGraceFromErrors != nil {
		wrappedError := errors.Wrap(updateGraceFromErrors, "failed to update grace_from for central instances")
		encounteredErrors = append(encounteredErrors, wrappedError)
	}
	return encounteredErrors
}

func (k *GracePeriodManager) reconcileCentralGraceFrom(centrals dbapi.CentralList) serviceErr.ErrorList {
	glog.Infof("reconciling grace period start date for central instances")
	var svcErrors serviceErr.ErrorList
	subscriptionStatusByOrg := map[string]bool{}

	for _, central := range centrals {
		glog.Infof("reconciling grace_from for central instance %q", central.ID)

		// skip update when Central is marked for deletion or is already being deleted
		if arrays.Contains(constants.GetDeletingStatuses(), central.Status) {
			glog.Infof("central %q is in %q state, skipping grace_from reconciliation", central.ID, central.Status)
			continue
		}

		glog.Infof("checking quota entitlement status for Central instance %q", central.ID)
		active, ok := subscriptionStatusByOrg[central.OrganisationID]
		if !ok {
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
		return k.updateCentralGraceDate(central, nil)
	}

	// if quota entitlement is not active and grace_from is not already set, set its value based on the current time and grace period allowance
	if !isQuotaEntitlementActive && central.GraceFrom == nil {
		// set grace_from to now + grace period days
		graceFromTime := time.Now()
		glog.Infof("quota entitlement for central instance %q is no longer active, updating grace_from to %q", central.ID, graceFromTime.Format(time.RFC1123Z))
		return k.updateCentralGraceDate(central, &graceFromTime)
	}

	glog.Infof("no grace_from changes needed for central %q, skipping update", central.ID)
	return nil
}

// updates the grace_from field for the given Central instance
func (k *GracePeriodManager) updateCentralGraceDate(central *dbapi.CentralRequest, graceFromTime *time.Time) error {
	var graceFrom sql.NullTime
	if graceFromTime != nil {
		graceFrom = sql.NullTime{Time: *graceFromTime, Valid: true}
	}

	if err := k.dinosaurService.Updates(central, map[string]interface{}{
		"grace_from": graceFrom,
	}); err != nil {
		return err
	}

	return nil
}
