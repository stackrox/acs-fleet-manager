package workers

import (
	"fmt"
	"time"

	"github.com/goava/di"

	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/golang/glog"
)

// Reconciler ...
type Reconciler struct {
	di.Inject
}

// Start ...
func (r *Reconciler) Start(worker Worker) {
	*worker.GetStopChan() = make(chan struct{})
	worker.GetSyncGroup().Add(1)
	worker.SetIsRunning(true)

	ticker := time.NewTicker(worker.GetRepeatInterval())
	go func() {
		// starts reconcile immediately and then on every repeat interval
		glog.V(1).Infoln(fmt.Sprintf("Initial reconciliation loop for %T [%s]", worker, worker.GetID()))
		r.runReconcile(worker)
		for {
			select {
			case <-ticker.C:
				r.runReconcile(worker)
			case <-*worker.GetStopChan():
				ticker.Stop()
				defer worker.GetSyncGroup().Done()
				glog.V(1).Infoln(fmt.Sprintf("Stopping reconciliation loop for %T [%s]", worker, worker.GetID()))
				return
			}
		}
	}()
}

func (r *Reconciler) runReconcile(worker Worker) {
	start := time.Now()
	errors := worker.Reconcile()
	if len(errors) == 0 {
		metrics.IncreaseReconcilerSuccessCount(worker.GetWorkerType())
	} else {
		metrics.IncreaseReconcilerFailureCount(worker.GetWorkerType())
		metrics.IncreaseReconcilerErrorsCount(worker.GetWorkerType(), len(errors))
	}
	metrics.UpdateReconcilerDurationMetric(worker.GetWorkerType(), time.Since(start))
	for _, e := range errors {
		logger.Logger.Error(e)
	}
}

// Stop ...
func (r *Reconciler) Stop(worker Worker) {
	defer worker.SetIsRunning(false)
	select {
	case <-*worker.GetStopChan(): // already closed
		return
	default:
		close(*worker.GetStopChan()) // explicit close
		worker.GetSyncGroup().Wait() // wait for in-flight job to finish
	}
	metrics.ResetMetricsForReconcilers()
}
