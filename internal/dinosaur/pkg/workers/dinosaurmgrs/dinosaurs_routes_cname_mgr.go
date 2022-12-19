package dinosaurmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// DinosaurRoutesCNAMEManager ...
type DinosaurRoutesCNAMEManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
	dinosaurConfig  *config.CentralConfig
}

var _ workers.Worker = &DinosaurRoutesCNAMEManager{}

// NewDinosaurCNAMEManager ...
func NewDinosaurCNAMEManager(dinosaurService services.DinosaurService, kafkfConfig *config.CentralConfig) *DinosaurRoutesCNAMEManager {
	return &DinosaurRoutesCNAMEManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "dinosaur_dns",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService: dinosaurService,
		dinosaurConfig:  kafkfConfig,
	}
}

// Start ...
func (k *DinosaurRoutesCNAMEManager) Start() {
	k.StartWorker(k)
}

// Stop ...
func (k *DinosaurRoutesCNAMEManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *DinosaurRoutesCNAMEManager) Reconcile() []error {
	glog.Infoln("reconciling DNS for centrals")
	var errs []error

	dinosaurs, listErr := k.dinosaurService.ListDinosaursWithRoutesNotCreated()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list centrals whose routes are not created"))
	} else {
		glog.Infof("centrals need routes created count = %d", len(dinosaurs))
	}

	for _, dinosaur := range dinosaurs {
		if k.dinosaurConfig.EnableCentralExternalCertificate {
			if dinosaur.RoutesCreationID == "" {
				glog.Infof("creating CNAME records for central %s", dinosaur.ID)

				changeOutput, err := k.dinosaurService.ChangeDinosaurCNAMErecords(dinosaur, services.DinosaurRoutesActionCreate)

				if err != nil {
					errs = append(errs, err)
					continue
				}

				if changeOutput.ChangeInfo == nil {
					errs = append(errs, errors.Wrapf(listErr, "failed to get ChangeInfo for %s, %s", dinosaur.ID, dinosaur.Name))
				} else {
					dinosaur.RoutesCreationID = *changeOutput.ChangeInfo.Id
					dinosaur.RoutesCreated = *changeOutput.ChangeInfo.Status == "INSYNC"
				}
			} else {
				recordStatus, err := k.dinosaurService.GetCNAMERecordStatus(dinosaur)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				dinosaur.RoutesCreated = *recordStatus.Status == "INSYNC"
			}
		} else {
			glog.Infof("external certificate is disabled, skip CNAME creation for Central %s", dinosaur.ID)
			dinosaur.RoutesCreated = true
		}

		if err := k.dinosaurService.Update(dinosaur); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
