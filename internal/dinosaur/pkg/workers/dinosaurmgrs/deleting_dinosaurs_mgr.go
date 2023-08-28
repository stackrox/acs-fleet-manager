package dinosaurmgrs

import (
	"context"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	dynamicClientAPI "github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/golang/glog"
)

const deletingCentralWorkerType = "deleting_dinosaur"

// DeletingDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests.
type DeletingDinosaurManager struct {
	workers.BaseWorker
	dinosaurService     services.DinosaurService
	iamConfig           *iam.IAMConfig
	quotaServiceFactory services.QuotaServiceFactory
	dynamicAPI          *dynamicClientAPI.AcsTenantsApiService
}

// NewDeletingDinosaurManager creates a new dinosaur manager.
func NewDeletingDinosaurManager(dinosaurService services.DinosaurService, iamConfig *iam.IAMConfig,
	quotaServiceFactory services.QuotaServiceFactory) *DeletingDinosaurManager {
	metrics.InitReconcilerMetricsForType(deletingCentralWorkerType)
	return &DeletingDinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: deletingCentralWorkerType,
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:     dinosaurService,
		iamConfig:           iamConfig,
		dynamicAPI:          dynamicclients.NewDynamicClientsAPI(iamConfig.RedhatSSORealm),
		quotaServiceFactory: quotaServiceFactory,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests.
func (k *DeletingDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *DeletingDinosaurManager) Stop() {
	k.StopWorker(k)
}

// Reconcile reconciles deleting dionosaur requests.
// It handles:
//   - freeing up any associated quota with the central
//   - any dynamically created OIDC client within sso.redhat.com
func (k *DeletingDinosaurManager) Reconcile() []error {
	var encounteredErrors []error

	// handle deleting dinosaur requests
	// Dinosaurs in a "deleting" state have been removed, along with all their resources (i.e. ManagedDinosaur, Dinosaur CRs),
	// from the data plane cluster by the Fleetshard operator. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these Dinosaurs and their records soft deleted from the database.
	deletingDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants.CentralRequestStatusDeleting)
	originalTotalDinosaurInDeleting := len(deletingDinosaurs)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting central requests"))
	}
	if originalTotalDinosaurInDeleting > 0 {
		glog.Infof("%s centrals count = %d", constants.CentralRequestStatusDeleting.String(), originalTotalDinosaurInDeleting)
	}

	// We also want to remove Dinosaurs that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants.CentralRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list central deprovisioning requests"))
	}
	if len(deprovisioningDinosaurs) > 0 {
		glog.Infof("%s centrals count = %d", constants.CentralRequestStatusDeprovision.String(), len(deprovisioningDinosaurs))
	}

	for _, deprovisioningDinosaur := range deprovisioningDinosaurs {
		glog.V(10).Infof("deprovision central id = %s", deprovisioningDinosaur.ID)
		// TODO check if a deprovisioningDinosaur can be deleted and add it to deletingDinosaurs array
		// deletingDinosaurs = append(deletingDinosaurs, deprovisioningDinosaur)
		if deprovisioningDinosaur.Host == "" {
			deletingDinosaurs = append(deletingDinosaurs, deprovisioningDinosaur)
		}
	}

	additionalMarkedRemovals := len(deletingDinosaurs) - originalTotalDinosaurInDeleting
	if additionalMarkedRemovals > 0 {
		glog.Infof("An additional of centrals count = %d which are marked for removal before being provisioned will also be deleted", additionalMarkedRemovals)
	}

	for _, dinosaur := range deletingDinosaurs {
		glog.V(10).Infof("deleting central id = %s", dinosaur.ID)
		if err := k.reconcileDeletingDinosaurs(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile deleting central request %s", dinosaur.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *DeletingDinosaurManager) reconcileDeletingDinosaurs(dinosaur *dbapi.CentralRequest) error {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(dinosaur.QuotaType))
	if factoryErr != nil {
		return factoryErr
	}
	err := quotaService.DeleteQuota(dinosaur.SubscriptionID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete subscription id %s for central %s", dinosaur.SubscriptionID, dinosaur.ID)
	}

	switch dinosaur.ClientOrigin {
	case dbapi.AuthConfigStaticClientOrigin:
		glog.V(7).Infof("central %s uses static client; no dynamic client will be attempted to be deleted",
			dinosaur.ID)
	case dbapi.AuthConfigDynamicClientOrigin:
		if resp, err := k.dynamicAPI.DeleteAcsClient(context.Background(), dinosaur.ClientID); err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				glog.V(7).Infof("dynamic client %s could not be found; will continue as if the client "+
					"has been deleted", dinosaur.ClientID)
			} else {
				return errors.Wrapf(err, "failed to delete dynamic OIDC client id %s for central %s",
					dinosaur.ClientID, dinosaur.ID)
			}
		}
	default:
		glog.V(1).Infof("invalid client origin %s found for central %s. No deletion will be attempted",
			dinosaur.ClientOrigin, dinosaur.ID)
	}

	if err := k.dinosaurService.Delete(dinosaur, false); err != nil {
		return errors.Wrapf(err, "failed to delete central %s", dinosaur.ID)
	}
	return nil
}
