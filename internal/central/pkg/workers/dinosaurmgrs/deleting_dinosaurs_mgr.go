package dinosaurmgrs

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/golang/glog"
)

// DeletingCentralManager represents a dinosaur manager that periodically reconciles dinosaur requests
type DeletingCentralManager struct {
	workers.BaseWorker
	centralService      services.CentralService
	iamConfig           *iam.IAMConfig
	quotaServiceFactory services.QuotaServiceFactory
}

// NewDeletingCentralManager creates a new dinosaur manager
func NewDeletingCentralManager(centralService services.CentralService, iamConfig *iam.IAMConfig, quotaServiceFactory services.QuotaServiceFactory) *DeletingCentralManager {
	return &DeletingCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "deleting_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		centralService:      centralService,
		iamConfig:           iamConfig,
		quotaServiceFactory: quotaServiceFactory,
	}
}

// Start initializes the central manager to reconcile central requests
func (k *DeletingCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *DeletingCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *DeletingCentralManager) Reconcile() []error {
	glog.Infoln("reconciling deleting centrals")
	var encounteredErrors []error

	// handle deleting central requests
	// centrals in a "deleting" state have been removed, along with all their resources (i.e. ManagedCentral, central CRs),
	// from the data plane cluster by the Fleetshard operator. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these centrals and their records soft deleted from the database.

	deletingCentrals, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusDeleting)
	originalTotalCentralInDeleting := len(deletingCentrals)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting central requests"))
	} else {
		glog.Infof("%s centrals count = %d", constants2.CentralRequestStatusDeleting.String(), originalTotalCentralInDeleting)
	}

	// We also want to remove Dinosaurs that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningCentrals, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list central deprovisioning requests"))
	} else {
		glog.Infof("%s centrals count = %d", constants2.CentralRequestStatusDeprovision.String(), len(deprovisioningCentrals))
	}

	for _, deprovisioningCentral := range deprovisioningCentrals {
		glog.V(10).Infof("deprovision central id = %s", deprovisioningCentral.ID)
		// TODO check if a deprovisioningCentral can be deleted and add it to deletingCentrals array
		// deletingCentrals = append(deletingCentrals, deprovisioningCentral)
		if deprovisioningCentral.Host == "" {
			deletingCentrals = append(deletingCentrals, deprovisioningCentral)
		}
	}

	glog.Infof("An additional of centrals count = %d which are marked for removal before being provisioned will also be deleted", len(deletingCentrals)-originalTotalCentralInDeleting)

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

	if err := k.centralService.Delete(central); err != nil {
		return errors.Wrapf(err, "failed to delete central %s", central.ID)
	}
	return nil
}
