package centralmgrs

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

const readyCentralWorkerType = "ready_central"

var readyCentralCountCache int32

// ReadyCentralManager represents a central manager that periodically reconciles central requests
type ReadyCentralManager struct {
	workers.BaseWorker
	centralService services.CentralService
}

// NewReadyCentralManager creates a new central manager
func NewReadyCentralManager(centralService services.CentralService) *ReadyCentralManager {
	metrics.InitReconcilerMetricsForType(readyCentralWorkerType)
	return &ReadyCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: readyCentralWorkerType,
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
	}
}

// Start initializes the central manager to reconcile central requests
func (k *ReadyCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *ReadyCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *ReadyCentralManager) Reconcile() []error {
	var encounteredErrors []error

	readyCentrals, serviceErr := k.centralService.ListByStatus(constants.CentralRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready centrals"))
	}
	readyCentralCountCache = int32(len(readyCentrals))
	logger.InfoChangedInt32(&readyCentralCountCache, "ready centrals count = %d", readyCentralCountCache)

	return encounteredErrors
}
