package centralmgrs

import (
	"context"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	dynamicClientAPI "github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/golang/glog"
)

const deletingCentralWorkerType = "deleting_central"

// DeletingCentralManager represents a central manager that periodically reconciles central requests.
type DeletingCentralManager struct {
	workers.BaseWorker
	centralService      services.CentralService
	iamConfig           *iam.IAMConfig
	quotaServiceFactory services.QuotaServiceFactory
	dynamicAPI          *dynamicClientAPI.AcsTenantsApiService
}

// NewDeletingCentralManager creates a new central manager.
func NewDeletingCentralManager(centralService services.CentralService, iamConfig *iam.IAMConfig,
	quotaServiceFactory services.QuotaServiceFactory) *DeletingCentralManager {
	metrics.InitReconcilerMetricsForType(deletingCentralWorkerType)
	return &DeletingCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: deletingCentralWorkerType,
			Reconciler: workers.Reconciler{},
		},
		centralService:      centralService,
		iamConfig:           iamConfig,
		dynamicAPI:          dynamicclients.NewDynamicClientsAPI(iamConfig.RedhatSSORealm),
		quotaServiceFactory: quotaServiceFactory,
	}
}

// Start initializes the central manager to reconcile central requests.
func (k *DeletingCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *DeletingCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile reconciles deleting dionosaur requests.
// It handles:
//   - freeing up any associated quota with the central
//   - any dynamically created OIDC client within sso.redhat.com
func (k *DeletingCentralManager) Reconcile() []error {
	var encounteredErrors []error

	// handle deleting central requests
	// Centrals in a "deleting" state have been removed, along with all their resources (i.e. ManagedCentral, Central CRs),
	// from the data plane cluster by Fleetshard Sync. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these Centrals and their records soft deleted from the database.
	deletingCentrals, serviceErr := k.centralService.ListByStatus(constants.CentralRequestStatusDeleting)
	originalTotalCentralInDeleting := len(deletingCentrals)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting central requests"))
	}
	if originalTotalCentralInDeleting > 0 {
		glog.Infof("%s centrals count = %d", constants.CentralRequestStatusDeleting.String(), originalTotalCentralInDeleting)
	}

	// We also want to remove Centrals that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningCentrals, serviceErr := k.centralService.ListByStatus(constants.CentralRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list central deprovisioning requests"))
	}
	if len(deprovisioningCentrals) > 0 {
		glog.Infof("%s centrals count = %d", constants.CentralRequestStatusDeprovision.String(), len(deprovisioningCentrals))
	}

	for _, deprovisioningCentral := range deprovisioningCentrals {
		glog.V(10).Infof("deprovision central id = %s", deprovisioningCentral.ID)
		// TODO check if a deprovisioningCentral can be deleted and add it to deletingCentrals array
		// deletingCentrals = append(deletingCentrals, deprovisioningCentral)
		if deprovisioningCentral.Host == "" {
			deletingCentrals = append(deletingCentrals, deprovisioningCentral)
		}
	}

	additionalMarkedRemovals := len(deletingCentrals) - originalTotalCentralInDeleting
	if additionalMarkedRemovals > 0 {
		glog.Infof("An additional of centrals count = %d which are marked for removal before being provisioned will also be deleted", additionalMarkedRemovals)
	}

	for _, central := range deletingCentrals {
		glog.V(10).Infof("deleting central id = %s", central.ID)
		if err := k.reconcileDeletingCentrals(central); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile deleting central request %s", central.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *DeletingCentralManager) reconcileDeletingCentrals(central *dbapi.CentralRequest) error {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(central.QuotaType))
	if factoryErr != nil {
		return factoryErr
	}
	err := quotaService.DeleteQuota(central.SubscriptionID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete subscription id %s for central %s", central.SubscriptionID, central.ID)
	}

	switch central.ClientOrigin {
	case dbapi.AuthConfigStaticClientOrigin:
		glog.V(7).Infof("central %s uses static client; no dynamic client will be attempted to be deleted",
			central.ID)
	case dbapi.AuthConfigDynamicClientOrigin:
		if resp, err := k.dynamicAPI.DeleteAcsClient(context.Background(), central.ClientID); err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				glog.V(7).Infof("dynamic client %s could not be found; will continue as if the client "+
					"has been deleted", central.ClientID)
			} else {
				return errors.Wrapf(err, "failed to delete dynamic OIDC client id %s for central %s",
					central.ClientID, central.ID)
			}
		}
	default:
		glog.V(1).Infof("invalid client origin %s found for central %s. No deletion will be attempted",
			central.ClientOrigin, central.ID)
	}

	if err := k.centralService.Delete(central, false); err != nil {
		return errors.Wrapf(err, "failed to delete central %s", central.ID)
	}
	return nil
}
