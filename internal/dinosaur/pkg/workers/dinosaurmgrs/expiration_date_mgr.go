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

// ExpirationDateManager represents a central manager that manages the expiration date.
type ExpirationDateManager struct {
	workers.BaseWorker
	centralService      services.DinosaurService
	quotaServiceFactory services.QuotaServiceFactory
	centralConfig       *config.CentralConfig
}

// NewExpirationDateManager creates a new grace period manager
func NewExpirationDateManager(centralService services.DinosaurService, quotaServiceFactory services.QuotaServiceFactory, centralConfig *config.CentralConfig) *ExpirationDateManager {
	return &ExpirationDateManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "expiration_date_worker",
			Reconciler: workers.Reconciler{},
		},
		centralService:      centralService,
		quotaServiceFactory: quotaServiceFactory,
		centralConfig:       centralConfig,
	}
}

// GetRepeatInterval doesn't need to be frequent for this worker.
func (*ExpirationDateManager) GetRepeatInterval() time.Duration {
	return 6 * time.Hour
}

// Start initializes the central manager to reconcile central requests
func (k *ExpirationDateManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *ExpirationDateManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *ExpirationDateManager) Reconcile() []error {
	glog.Infoln("reconciling grace period start date for central instances")
	var encounteredErrors []error

	centrals, svcErr := k.centralService.ListByStatus(constants.ActiveStatuses...)
	if svcErr != nil {
		return append(encounteredErrors, svcErr)
	}

	// reconciles expired_at field for central instances
	updateExpiredAtErrors := k.reconcileCentralExpiredAt(centrals)
	if updateExpiredAtErrors != nil {
		wrappedError := errors.Wrap(updateExpiredAtErrors, "failed to update expired_at for central instances")
		encounteredErrors = append(encounteredErrors, wrappedError)
	}
	return encounteredErrors
}

func (k *ExpirationDateManager) reconcileCentralExpiredAt(centrals dbapi.CentralList) serviceErr.ErrorList {
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

		if err := k.updateExpiredAtBasedOnQuotaEntitlement(central, defined); err != nil {
			svcErrors = append(svcErrors, errors.Wrapf(err, "failed to update expired_at value based on quota entitlement for central instance %q", central.ID))
		}
	}

	return svcErrors
}

// Updates expired_at field of the given Central instance based on the user/organisation's quota entitlement status
func (k *ExpirationDateManager) updateExpiredAtBasedOnQuotaEntitlement(central *dbapi.CentralRequest, isQuotaEntitlementActive bool) *serviceErr.ServiceError {
	// if quota entitlement is active, ensure expired_at is set to null.
	if isQuotaEntitlementActive && central.ExpiredAt != nil {
		central.ExpiredAt = nil
		glog.Infof("updating grace start date of central instance %q to NULL", central.ID)
		return k.centralService.Update(central)
	}

	// if quota entitlement is not active and expired_at is not already set, set
	// its value to the current time.
	if !isQuotaEntitlementActive && central.ExpiredAt == nil {
		now := time.Now()
		central.ExpiredAt = &now
		glog.Infof("quota entitlement for central instance %q is no longer active, updating expired_at to %q", central.ID, now.Format(time.RFC1123Z))
		return k.centralService.Update(central)
	}
	return nil
}
