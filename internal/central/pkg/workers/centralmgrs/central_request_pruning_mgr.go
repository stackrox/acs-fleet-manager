package centralmgrs

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

const (
	centralRequestPruningWorkerType = "central_request_pruning"

	standardRetention = 2 * 365 * 24 * time.Hour // 2 years
	internalRetention = 14 * 24 * time.Hour      // 14 days
)

// CentralRequestPruningManager permanently deletes soft-deleted central_requests
// that have exceeded their retention period.
type CentralRequestPruningManager struct {
	workers.BaseWorker
	connectionFactory *db.ConnectionFactory
}

// NewCentralRequestPruningManager creates a new pruning manager.
func NewCentralRequestPruningManager(connectionFactory *db.ConnectionFactory) *CentralRequestPruningManager {
	metrics.InitReconcilerMetricsForType(centralRequestPruningWorkerType)
	return &CentralRequestPruningManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: centralRequestPruningWorkerType,
			Reconciler: workers.Reconciler{},
		},
		connectionFactory: connectionFactory,
	}
}

// GetRepeatInterval returns how often the pruning worker runs.
func (*CentralRequestPruningManager) GetRepeatInterval() time.Duration {
	return 6 * time.Hour
}

// Start initializes the pruning worker.
func (m *CentralRequestPruningManager) Start() {
	m.StartWorker(m)
}

// Stop causes the pruning worker to stop.
func (m *CentralRequestPruningManager) Stop() {
	m.StopWorker(m)
}

// Reconcile permanently deletes soft-deleted central requests past their retention period.
func (m *CentralRequestPruningManager) Reconcile() []error {
	glog.Infoln("reconciling central_requests pruning")
	var errs []error

	if err := m.pruneStandard(); err != nil {
		errs = append(errs, err)
	}
	if err := m.pruneInternal(); err != nil {
		errs = append(errs, err)
	}

	return errs
}

func (m *CentralRequestPruningManager) pruneStandard() error {
	cutoff := time.Now().Add(-standardRetention)
	dbConn := m.connectionFactory.New()
	result := dbConn.Unscoped().
		Where("deleted_at IS NOT NULL").
		Where("deleted_at < ?", cutoff).
		Where("internal = false OR internal IS NULL").
		Where("status IN ?", terminalStatuses).
		Delete(&dbapi.CentralRequest{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		glog.Infof("pruned %d standard central requests older than %s", result.RowsAffected, standardRetention)
	}
	return nil
}

func (m *CentralRequestPruningManager) pruneInternal() error {
	cutoff := time.Now().Add(-internalRetention)
	dbConn := m.connectionFactory.New()
	result := dbConn.Unscoped().
		Where("deleted_at IS NOT NULL").
		Where("deleted_at < ?", cutoff).
		Where("internal = true").
		Where("status IN ?", terminalStatuses).
		Delete(&dbapi.CentralRequest{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		glog.Infof("pruned %d internal central requests older than %s", result.RowsAffected, internalRetention)
	}
	return nil
}

var terminalStatuses = []string{
	constants.CentralRequestStatusDeprovision.String(),
	constants.CentralRequestStatusDeleting.String(),
	constants.CentralRequestStatusFailed.String(),
}
