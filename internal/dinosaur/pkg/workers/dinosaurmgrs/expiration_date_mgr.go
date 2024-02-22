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
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

const leaseType = "expiration_date_worker"

// ExpirationDateManager set's the `expired_at` central request property to the
// current time when the quota allowance returned from AMS equals to 0.
type ExpirationDateManager struct {
	workers.BaseWorker
	centralService      services.DinosaurService
	quotaServiceFactory services.QuotaServiceFactory
	centralConfig       *config.CentralConfig
}

// NewExpirationDateManager creates a new expiration date manager.
func NewExpirationDateManager(centralService services.DinosaurService, quotaServiceFactory services.QuotaServiceFactory, centralConfig *config.CentralConfig) *ExpirationDateManager {
	return &ExpirationDateManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: leaseType,
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

// Start initializes the central manager to reconcile central requests.
func (k *ExpirationDateManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *ExpirationDateManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *ExpirationDateManager) Reconcile() []error {
	glog.Infoln("reconciling expiration date for central instances")
	var encounteredErrors []error

	centrals, svcErr := k.centralService.ListByStatus(
		append(constants.ActiveStatuses, constants.CentralRequestStatusFailed)...)
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

	quotaCostCache := make(map[quotaCostCacheKey]bool, 0)
	for _, central := range centrals {
		key := quotaCostCacheKey{central.OrganisationID, central.CloudAccountID, central.InstanceType}
		active, inCache := quotaCostCache[key]
		if !inCache {
			var svcErr *serviceErr.ServiceError
			active, svcErr = quotaService.HasQuotaAllowance(central, types.DinosaurInstanceType(central.InstanceType))
			if svcErr != nil {
				svcErrors = append(svcErrors, errors.Wrapf(svcErr, "failed to get quota entitlement status of central instance %q", central.ID))
				continue
			}
			quotaCostCache[key] = active
		}

		if timestamp, needsChange := k.expiredAtNeedsUpdate(central, active); needsChange {
			central.ExpiredAt = timestamp
			if err := k.updateExpiredAtInDB(central); err != nil {
				svcErrors = append(svcErrors, errors.Wrapf(err,
					"failed to update expired_at value based on quota entitlement for central instance %q", central.ID))
			}
		}
	}

	return svcErrors
}

func (k *ExpirationDateManager) updateExpiredAtInDB(central *dbapi.CentralRequest) *serviceErr.ServiceError {
	glog.Infof("updating expired_at of central %q to %q", central.ID, central.ExpiredAt)
	if central.ExpiredAt != nil {
		metrics.CentralExpirationSet(central.ID)
	}
	return k.centralService.Updates(&dbapi.CentralRequest{Meta: api.Meta{ID: central.ID}},
		map[string]interface{}{"expired_at": central.ExpiredAt})
}

// Returns whether the expired_at field of the given Central instance needs to be updated.
func (k *ExpirationDateManager) expiredAtNeedsUpdate(central *dbapi.CentralRequest, isQuotaEntitlementActive bool) (*time.Time, bool) {
	// if quota entitlement is active, ensure expired_at is set to null.
	if isQuotaEntitlementActive && central.ExpiredAt != nil {
		return nil, true
	}

	// if quota entitlement is not active and expired_at is not already set, set
	// its value to the current time.
	if !isQuotaEntitlementActive && central.ExpiredAt == nil {
		now := time.Now()
		return &now, true
	}
	return nil, false
}
