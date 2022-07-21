package dinosaur_mgrs

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
	dinosaurConfig  *config.DinosaurConfig
}

var _ workers.Worker = &DinosaurRoutesCNAMEManager{}

// NewDinosaurCNAMEManager ...
func NewDinosaurCNAMEManager(dinosaurService services.DinosaurService, kafkfConfig *config.DinosaurConfig) *DinosaurRoutesCNAMEManager {
	return &DinosaurRoutesCNAMEManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
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
	glog.Infoln("reconciling DNS for dinosaurs")
	var errs []error

	dinosaurs, listErr := k.dinosaurService.ListDinosaursWithRoutesNotCreated()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list dinosaurs whose routes are not created"))
	} else {
		glog.Infof("dinosaurs need routes created count = %d", len(dinosaurs))
	}

	for _, dinosaur := range dinosaurs {
		if k.dinosaurConfig.EnableDinosaurExternalCertificate {
			if dinosaur.RoutesCreationId == "" {
				glog.Infof("creating CNAME records for dinosaur %s", dinosaur.ID)

				changeOutput, err := k.dinosaurService.ChangeDinosaurCNAMErecords(dinosaur, services.DinosaurRoutesActionCreate)

				if err != nil {
					errs = append(errs, err)
					continue
				}

				dinosaur.RoutesCreationId = *changeOutput.ChangeInfo.Id
				dinosaur.RoutesCreated = *changeOutput.ChangeInfo.Status == "INSYNC"
			} else {
				recordStatus, err := k.dinosaurService.GetCNAMERecordStatus(dinosaur)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				dinosaur.RoutesCreated = *recordStatus.Status == "INSYNC"
			}
		} else {
			glog.Infof("external certificate is disabled, skip CNAME creation for Dinosaur %s", dinosaur.ID)
			dinosaur.RoutesCreated = true
		}

		if err := k.dinosaurService.Update(dinosaur); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
