package workers

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestReconciler_Wakeup(t *testing.T) {
	RegisterTestingT(t)
	r := Reconciler{}
	var stopchan chan struct{}
	var wg sync.WaitGroup

	reconcileChan := make(chan time.Time, 1000)
	worker := &WorkerMock{
		GetStopChanFunc: func() *chan struct{} {
			return &stopchan
		},
		GetSyncGroupFunc: func() *sync.WaitGroup {
			return &wg
		},
		SetIsRunningFunc: func(val bool) {
		},
		GetIDFunc: func() string {
			return "test"
		},
		GetWorkerTypeFunc: func() string {
			return "test"
		},
		ReconcileFunc: func() []error {
			var errors []error
			reconcileChan <- time.Now()
			return errors
		},
	}

	waitForReconcile := func(d time.Duration) (timeout bool) {
		if d == 0 {
			select {
			case <-reconcileChan:
			default:
				timeout = true
			}
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()
			select {
			case <-reconcileChan:
			case <-ctx.Done():
				timeout = true
			}
		}
		return
	}

	r.Start(worker)
	defer r.Stop(worker)

	// initial reconcile should happen right away... this should not timeout
	Expect(waitForReconcile(1 * time.Second)).Should(Equal(false))

	// Next reconcile will take a while since it runs every 30 seconds.. lets timeout after 3 seconds of waiting..
	Expect(waitForReconcile(3 * time.Second)).Should(Equal(true))

	// Now lets try to wake it up before those 30 seconds have passed...
	r.Wakeup(false)
	Expect(waitForReconcile(1 * time.Second)).Should(Equal(false))

	r.Wakeup(true)
	// We can use a 0 timeout here because Wakeup will wait for the reconcile to occur first.
	Expect(waitForReconcile(0)).Should(Equal(false))
}
