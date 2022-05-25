package dinosaur_mgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// ReadyDinosaurManager represents a dinosaur manager that periodically reconciles dinosaur requests
type ReadyDinosaurManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
	keycloakService sso.KeycloakService
	keycloakConfig  *keycloak.KeycloakConfig
}

// NewReadyDinosaurManager creates a new dinosaur manager
func NewReadyDinosaurManager(dinosaurService services.DinosaurService, keycloakService sso.DinosaurKeycloakService, keycloakConfig *keycloak.KeycloakConfig) *ReadyDinosaurManager {
	return &ReadyDinosaurManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "ready_dinosaur",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService: dinosaurService,
		keycloakService: keycloakService,
		keycloakConfig:  keycloakConfig,
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

func (k *ReadyDinosaurManager) Reconcile() []error {
	glog.Infoln("reconciling ready dinosaurs")

	var encounteredErrors []error

	readyDinosaurs, serviceErr := k.dinosaurService.ListByStatus(constants2.DinosaurRequestStatusReady)
	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list ready dinosaurs"))
	} else {
		glog.Infof("ready dinosaurs count = %d", len(readyDinosaurs))
	}

	for _, dinosaur := range readyDinosaurs {
		glog.V(10).Infof("ready dinosaur id = %s", dinosaur.ID)
		// TODO implement reconciliation logic for ready dinosaurs
	}

	return encounteredErrors
}
