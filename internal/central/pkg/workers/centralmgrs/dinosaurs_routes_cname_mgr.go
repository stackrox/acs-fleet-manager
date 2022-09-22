package centralmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// CentralRoutesCNAMEManager ...
type CentralRoutesCNAMEManager struct {
	workers.BaseWorker
	centralService services.CentralService
	centralConfig  *config.CentralConfig
}

var _ workers.Worker = &CentralRoutesCNAMEManager{}

// NewCentralCNAMEManager ...
func NewCentralCNAMEManager(centralService services.CentralService, centralConfig *config.CentralConfig) *CentralRoutesCNAMEManager {
	return &CentralRoutesCNAMEManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "dinosaur_dns",
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
		centralConfig:  centralConfig,
	}
}

// Start ...
func (k *CentralRoutesCNAMEManager) Start() {
	k.StartWorker(k)
}

// Stop ...
func (k *CentralRoutesCNAMEManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *CentralRoutesCNAMEManager) Reconcile() []error {
	glog.Infoln("reconciling DNS for centrals")
	var errs []error

	centrals, listErr := k.centralService.ListCentralsWithRoutesNotCreated()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list centrals whose routes are not created"))
	} else {
		glog.Infof("centrals need routes created count = %d", len(centrals))
	}

	for _, central := range centrals {
		if k.centralConfig.EnableCentralExternalCertificate {
			if central.RoutesCreationID == "" {
				glog.Infof("creating CNAME records for central %s", central.ID)

				changeOutput, err := k.centralService.ChangeCentralCNAMERecords(central, services.CentralRoutesActionCreate)

				if err != nil {
					errs = append(errs, err)
					continue
				}

				central.RoutesCreationID = *changeOutput.ChangeInfo.Id
				central.RoutesCreated = *changeOutput.ChangeInfo.Status == "INSYNC"
			} else {
				recordStatus, err := k.centralService.GetCNAMERecordStatus(central)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				central.RoutesCreated = *recordStatus.Status == "INSYNC"
			}
		} else {
			glog.Infof("external certificate is disabled, skip CNAME creation for Central %s", central.ID)
			central.RoutesCreated = true
		}

		if err := k.centralService.Update(central); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
