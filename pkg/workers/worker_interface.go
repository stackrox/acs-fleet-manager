package workers

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

// DefaultRepeatInterval is default interval with which workers Reconcile() method will be called.
// It is variable and not constant so that we could easily change this value in tests.
var DefaultRepeatInterval = 30 * time.Second

// Worker ...
//
//go:generate moq -out worker_interface_moq.go . Worker
type Worker interface {
	GetID() string
	GetWorkerType() string
	Start()
	Stop()
	Reconcile() []error
	GetStopChan() *chan struct{}
	GetSyncGroup() *sync.WaitGroup
	IsRunning() bool
	SetIsRunning(val bool)
	GetRepeatInterval() time.Duration
}

// BaseWorker ...
type BaseWorker struct {
	ID           string
	WorkerType   string
	Reconciler   Reconciler
	isRunning    bool
	imStop       chan struct{}
	syncTeardown sync.WaitGroup
}

// GetID ...
func (b *BaseWorker) GetID() string {
	return b.ID
}

// GetWorkerType ...
func (b *BaseWorker) GetWorkerType() string {
	return b.WorkerType
}

// GetStopChan ...
func (b *BaseWorker) GetStopChan() *chan struct{} {
	return &b.imStop
}

// GetSyncGroup ...
func (b *BaseWorker) GetSyncGroup() *sync.WaitGroup {
	return &b.syncTeardown
}

// IsRunning ...
func (b *BaseWorker) IsRunning() bool {
	return b.isRunning
}

// SetIsRunning ...
func (b *BaseWorker) SetIsRunning(val bool) {
	b.isRunning = val
}

// StartWorker ...
func (b *BaseWorker) StartWorker(w Worker) {
	metrics.SetLeaderWorkerMetric(b.WorkerType, true)
	b.Reconciler.Start(w)
}

// StopWorker ...
func (b *BaseWorker) StopWorker(w Worker) {
	glog.Infof("Stopping reconciling worker id = %s", b.ID)
	b.Reconciler.Stop(w)
	metrics.ResetMetricsForCentralManagers()
	metrics.SetLeaderWorkerMetric(b.WorkerType, false)
}

func (b *BaseWorker) GetRepeatInterval() time.Duration {
	return DefaultRepeatInterval
}
