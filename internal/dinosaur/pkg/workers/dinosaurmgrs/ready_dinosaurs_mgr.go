package dinosaurmgrs

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

const readyCentralWorkerType = "ready_dinosaur"

var readyCentralCountCache int32

// ReadyDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ReadyDinosaurManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
	iamService      sso.IAMService
	iamConfig       *iam.IAMConfig
}

// NewReadyDinosaurManager creates a new dinosaur manager
func NewReadyDinosaurManager(dinosaurService services.DinosaurService, iamService sso.IAMService, iamConfig *iam.IAMConfig) *ReadyDinosaurManager {
	metrics.InitReconcilerMetricsForType(readyCentralWorkerType)
	return &ReadyDinosaurManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: readyCentralWorkerType,
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
	var encounteredErrors []error

	readyCentrals, serviceErr := k.dinosaurService.ListByStatus(constants2.CentralRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready centrals"))
	}
	readyCentralCountCache = int32(len(readyCentrals))
	logger.InfoChangedInt32(&readyCentralCountCache, "ready centrals count = %d", readyCentralCountCache)

	return encounteredErrors
}
