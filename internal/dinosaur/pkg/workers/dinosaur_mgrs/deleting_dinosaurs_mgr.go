package dinosaur_mgrs

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	"github.com/stackrox/acs-fleet-manager/pkg/api"

	"github.com/golang/glog"
)

// DeletingDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type DeletingDinosaurManager struct {
	workers.BaseWorker
	dinosaurService     services.DinosaurService
	keycloakConfig      *iam.IAMConfig
	quotaServiceFactory services.QuotaServiceFactory
}

// NewDeletingDinosaurManager creates a new dinosaur manager
func NewDeletingDinosaurManager(dinosaurService services.DinosaurService, keycloakConfig *iam.IAMConfig, quotaServiceFactory services.QuotaServiceFactory) *DeletingDinosaurManager {
	return &DeletingDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "deleting_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService:     dinosaurService,
		keycloakConfig:      keycloakConfig,
		quotaServiceFactory: quotaServiceFactory,
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

func (k *DeletingDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling deleting dinosaurs")
	var encounteredErrors []error

	// handle deleting dinosaur requests
	// Dinosaurs in a "deleting" state have been removed, along with all their resources (i.e. ManagedDinosaur, Dinosaur CRs),
	// from the data plane cluster by the Fleetshard operator. This reconcile phase ensures that any other
	// dependencies (i.e. SSO clients, CNAME records) are cleaned up for these Dinosaurs and their records soft deleted from the database.

	deletingDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusDeleting)
	originalTotalDinosaurInDeleting := len(deletingDinosaurs)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list deleting dinosaur requests"))
	} else {
		glog.Infof("%s dinosaurs count = %d", constants2.DinosaurRequestStatusDeleting.String(), originalTotalDinosaurInDeleting)
	}

	// We also want to remove Dinosaurs that are set to deprovisioning but have not been provisioned on a data plane cluster
	deprovisioningDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusDeprovision)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list dinosaur deprovisioning requests"))
	} else {
		glog.Infof("%s dinosaurs count = %d", constants2.DinosaurRequestStatusDeprovision.String(), len(deprovisioningDinosaurs))
	}

	for _, deprovisioningDinosaur := range deprovisioningDinosaurs {
		glog.V(10).Infof("deprovision dinosaur id = %s", deprovisioningDinosaur.ID)
		// TODO check if a deprovisioningDinosaur can be deleted and add it to deletingDinosaurs array
		// deletingDinosaurs = append(deletingDinosaurs, deprovisioningDinosaur)
		if deprovisioningDinosaur.Host == "" {
			deletingDinosaurs = append(deletingDinosaurs, deprovisioningDinosaur)
		}
	}

	glog.Infof("An additional of dinosaurs count = %d which are marked for removal before being provisioned will also be deleted", len(deletingDinosaurs)-originalTotalDinosaurInDeleting)

	for _, dinosaur := range deletingDinosaurs {
		glog.V(10).Infof("deleting dinosaur id = %s", dinosaur.ID)
		if err := k.reconcileDeletingDinosaurs(dinosaur); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile deleting dinosaur request %s", dinosaur.ID))
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
	err := quotaService.DeleteQuota(dinosaur.SubscriptionId)
	if err != nil {
		return errors.Wrapf(err, "failed to delete subscription id %s for dinosaur %s", dinosaur.SubscriptionId, dinosaur.ID)
	}

	if err := k.dinosaurService.Delete(dinosaur); err != nil {
		return errors.Wrapf(err, "failed to delete dinosaur %s", dinosaur.ID)
	}
	return nil
}
