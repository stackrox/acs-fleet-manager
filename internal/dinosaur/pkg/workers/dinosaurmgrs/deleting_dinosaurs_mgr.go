package dinosaurmgrs

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/golang/glog"
)

// DeletingDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type DeletingDinosaurManager struct {
	workers.BaseWorker
	centralService      services.DinosaurService
	iamConfig           *iam.IAMConfig
	quotaServiceFactory services.QuotaServiceFactory
	dynamicAPI          dynamicclients.Client
	centralConfig       *config.CentralConfig
}

// NewDeletingDinosaurManager creates a new dinosaur manager
func NewDeletingDinosaurManager(centralService services.DinosaurService, iamConfig *iam.IAMConfig, quotaServiceFactory services.QuotaServiceFactory,
	centralConfig *config.CentralConfig) *DeletingDinosaurManager {
	return &DeletingDinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "deleting_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		centralService:      centralService,
		iamConfig:           iamConfig,
		quotaServiceFactory: quotaServiceFactory,
		dynamicAPI:          dynamicclients.NewDynamicClientsClient(iamConfig.RedhatSSORealm),
		centralConfig:       centralConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *DeletingDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *DeletingDinosaurManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *DeletingDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling deleting centrals")
	var encounteredErrors []error

	// handle deleting dinosaur requests
	// Dinosaurs in a "deleting" state have been removed, along with all their resources (i.e. ManagedDinosaur, Dinosaur CRs),
	// from the data plane cluster by the Fleetshard operator. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these Dinosaurs and their records soft deleted from the database.
	deletingDinosaurs, serviceErr := k.centralService.ListByStatus(constants.CentralRequestStatusDeleting)
	originalTotalDinosaurInDeleting := len(deletingDinosaurs)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting central requests"))
	} else {
		glog.Infof("%s centrals count = %d", constants2.CentralRequestStatusDeleting.String(), originalTotalDinosaurInDeleting)
	}

	// We also want to remove Dinosaurs that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningDinosaurs, serviceErr := k.centralService.ListByStatus(constants.CentralRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list central deprovisioning requests"))
	} else {
		glog.Infof("%s centrals count = %d", constants2.CentralRequestStatusDeprovision.String(), len(deprovisioningDinosaurs))
	}

	for _, deprovisioningDinosaur := range deprovisioningDinosaurs {
		glog.V(10).Infof("deprovision central id = %s", deprovisioningDinosaur.ID)
		// TODO check if a deprovisioningDinosaur can be deleted and add it to deletingDinosaurs array
		// deletingDinosaurs = append(deletingDinosaurs, deprovisioningDinosaur)
		if deprovisioningDinosaur.Host == "" {
			deletingDinosaurs = append(deletingDinosaurs, deprovisioningDinosaur)
		}
	}

	glog.Infof("An additional of centrals count = %d which are marked for removal before being provisioned will also be deleted", len(deletingDinosaurs)-originalTotalDinosaurInDeleting)

	for _, dinosaur := range deletingDinosaurs {
		glog.V(10).Infof("deleting central id = %s", dinosaur.ID)
		if err := k.reconcileDeletingDinosaurs(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile deleting central request %s", dinosaur.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *DeletingDinosaurManager) reconcileDeletingDinosaurs(central *dbapi.CentralRequest) error {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(central.QuotaType))
	if factoryErr != nil {
		return factoryErr
	}
	err := quotaService.DeleteQuota(central.SubscriptionID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete subscription id %s for central %s", central.SubscriptionID, central.ID)
	}

	if k.centralConfig.HasStaticAuth() {
		glog.V(7).Infoln("static config found; no dynamic client will be deleted")
	} else {
		if err := k.dynamicAPI.DeleteDynamicClient(central.ClientID); err != nil {
			return errors.Wrapf(err, "failed to delete dynamic OIDC client id %s for central %s",
				central.ClientID, central.ID)
		}
	}

	if err := k.centralService.Delete(central, false); err != nil {
		return errors.Wrapf(err, "failed to delete central %s", central.ID)
	}
	return nil
}
