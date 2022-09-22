package dinosaurmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// ReadyDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ReadyDinosaurManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
	iamService      sso.IAMService
	iamConfig       *iam.IAMConfig
}

// NewReadyDinosaurManager creates a new dinosaur manager
func NewReadyDinosaurManager(dinosaurService services.DinosaurService, iamService sso.IAMService, iamConfig *iam.IAMConfig) *ReadyDinosaurManager {
	return &ReadyDinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "ready_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService: dinosaurService,
		iamService:      iamService,
		iamConfig:       iamConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *ReadyDinosaurManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *ReadyDinosaurManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *ReadyDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling ready centrals")

	var encounteredErrors []error

	readyDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.CentralRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready centrals"))
	} else {
		glog.Infof("ready centrals count = %d", len(readyDinosaurs))
	}

	for _, dinosaur := range readyDinosaurs {
		glog.V(10).Infof("ready central id = %s", dinosaur.ID)
		// TODO implement reconciliation logic for ready dinosaurs
	}

	return encounteredErrors
}
