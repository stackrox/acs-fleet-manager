package dinosaurmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// ReadyCentralManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ReadyCentralManager struct {
	workers.BaseWorker
	centralService services.CentralService
	iamService     sso.IAMService
	iamConfig      *iam.IAMConfig
}

// NewReadyCentralManager creates a new dinosaur manager
func NewReadyCentralManager(centralService services.CentralService, iamService sso.IAMService, iamConfig *iam.IAMConfig) *ReadyCentralManager {
	return &ReadyCentralManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "ready_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
		iamService:     iamService,
		iamConfig:      iamConfig,
	}
}

// Start initializes the dinosaur manager to reconcile dinosaur requests
func (k *ReadyCentralManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling dinosaur requests to stop.
func (k *ReadyCentralManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *ReadyCentralManager) Reconcile() []error {
	glog.Infoln("reconciling ready centrals")

	var encounteredErrors []error

	readyCentrals, serviceErr := k.centralService.ListByStatus(constants2.CentralRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready centrals"))
	} else {
		glog.Infof("ready dinosaurs count = %d", len(readyCentrals))
	}

	for _, central := range readyCentrals {
		glog.V(10).Infof("ready central id = %s", central.ID)
		// TODO implement reconciliation logic for ready dinosaurs
	}

	return encounteredErrors
}
