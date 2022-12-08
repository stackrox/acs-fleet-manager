package dinosaurmgrs

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// CentralsTimeoutManager represents a manager that periodically reconciles central requests
type CentralsTimeoutManager struct {
	workers.BaseWorker
	centralService services.DinosaurService
	timeout        time.Duration
}

var _ workers.Worker = (*CentralsTimeoutManager)(nil)

// NewCentralsTimeoutManager creates a new manager
func NewCentralsTimeoutManager(centralService services.DinosaurService, centralConfig *config.CentralConfig) *CentralsTimeoutManager {
	return &CentralsTimeoutManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "timeout_central",
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
		timeout:        centralConfig.CentralRequestExpirationTimeout,
	}
}

// Start initializes the manager to reconcile central requests
func (k *CentralsTimeoutManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling central requests to stop.
func (k *CentralsTimeoutManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *CentralsTimeoutManager) Reconcile() []error {
	glog.Infoln("reconciling centrals reached timeout")
	var encounteredErrors []error

	// list central requests eligible to be cancelled
	lastCreatedAt := time.Now().Add(-k.timeout)
	timedOutCentralRequests, serviceErr := k.centralService.ListTimedOutCentrals(lastCreatedAt)

	if serviceErr != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(serviceErr, "failed to list timed out centrals"))
	} else {
		glog.Infof("timed out centrals count = %d", len(timedOutCentralRequests))
	}
	for _, centralRequest := range timedOutCentralRequests {
		glog.V(10).Infof("timed out central id = %s", centralRequest.ID)
		if err := k.reconcileTimedOutCentral(centralRequest); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile timed out central %s", centralRequest.ID))
			continue
		}
	}

	return encounteredErrors
}

func (k *CentralsTimeoutManager) reconcileTimedOutCentral(centralRequest *dbapi.CentralRequest) error {
	centralRequest.Status = constants2.CentralRequestStatusFailed.String()
	centralRequest.FailedReason = "Creation time went over the timeout. Interrupting central initialization."
	if err := k.centralService.Update(centralRequest); err != nil {
		return errors.Wrapf(err, "failed to update timed out central %s", centralRequest.ID)
	}
	metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
	metrics.IncreaseCentralTimeoutCountMetric(centralRequest.ID, centralRequest.ClusterID)
	return nil
}
