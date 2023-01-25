package workers

import (
	"sync"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

// Worker ...
//
//go:generate moq -out woker_interface_moq.go . Worker
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
	glog.Infof("started worker %q", w.GetWorkerType())
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
