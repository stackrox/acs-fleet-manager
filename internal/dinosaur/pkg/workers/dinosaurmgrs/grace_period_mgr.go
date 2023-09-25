package dinosaurmgrs

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	serviceErr "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// GracePeriodManager represents a central manager that manages grace period date.
type GracePeriodManager struct {
	workers.BaseWorker
	centralService      services.DinosaurService
	quotaServiceFactory services.QuotaServiceFactory
	centralConfig       *config.CentralConfig
}

// NewGracePeriodManager creates a new grace period manager
func NewGracePeriodManager(centralService services.DinosaurService, quotaServiceFactory services.QuotaServiceFactory, centralConfig *config.CentralConfig) *GracePeriodManager {
	return &GracePeriodManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "grace_period_worker",
			Reconciler: workers.Reconciler{},
		},
		centralService:      centralService,
		quotaServiceFactory: quotaServiceFactory,
		centralConfig:       centralConfig,
	}
}

// GetRepeatInterval doesn't need to be frequent for this worker.
func (*GracePeriodManager) GetRepeatInterval() time.Duration {
	return 6 * time.Hour
}

// Start initializes the central manager to reconcile central requests
func (k *GracePeriodManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *GracePeriodManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *GracePeriodManager) Reconcile() []error {
	glog.Infoln("reconciling grace period start date for central instances")
	var encounteredErrors []error

	centrals, svcErr := k.centralService.ListByStatus(
		constants.CentralRequestStatusAccepted,
		constants.CentralRequestStatusPreparing,
		constants.CentralRequestStatusProvisioning,
		constants.CentralRequestStatusReady)
	if svcErr != nil {
		return append(encounteredErrors, svcErr)
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
	var svcErrors serviceErr.ErrorList

	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.centralConfig.Quota.Type))
	if factoryErr != nil {
		return append(svcErrors, factoryErr)
	}

	type quotaCostCacheKey struct {
		orgID          string
		cloudAccountID string
		instanceType   string
	}

	quotaCostCache := make(map[quotaCostCacheKey]bool)
	for _, central := range centrals {
		key := quotaCostCacheKey{central.OrganisationID, central.CloudAccountID, central.InstanceType}
		defined, inCache := quotaCostCache[key]
		if !inCache {
			var svcErr *serviceErr.ServiceError
			defined, svcErr = quotaService.CheckIfQuotaIsDefinedForInstanceType(central, types.DinosaurInstanceType(central.InstanceType))
			if svcErr != nil {
				svcErrors = append(svcErrors, errors.Wrapf(svcErr, "failed to get quota entitlement status of central instance %q", central.ID))
				continue
			}
			quotaCostCache[key] = defined
		}

		if err := k.updateGraceFromBasedOnQuotaEntitlement(central, defined); err != nil {
			svcErrors = append(svcErrors, errors.Wrapf(err, "failed to update grace_from value based on quota entitlement for central instance %q", central.ID))
		}
	}

	return svcErrors
}

// Updates grace_from field of the given Central instance based on the user/organisation's quota entitlement status
func (k *GracePeriodManager) updateGraceFromBasedOnQuotaEntitlement(central *dbapi.CentralRequest, isQuotaEntitlementActive bool) *serviceErr.ServiceError {
	// if quota entitlement is active, ensure grace_from is set to null
	if isQuotaEntitlementActive && central.GraceFrom != nil {
		central.GraceFrom = nil
		glog.Infof("updating grace start date of central instance %q to NULL", central.ID)
		return k.centralService.Update(central)
	}

	// if quota entitlement is not active and grace_from is not already set, set its value based on the current time and grace period allowance
	if !isQuotaEntitlementActive && central.GraceFrom == nil {
		now := time.Now()
		central.GraceFrom = &now
		glog.Infof("quota entitlement for central instance %q is no longer active, updating grace_from to %q", central.ID, now.Format(time.RFC1123Z))
		return k.centralService.Update(central)
	}
	return nil
}
