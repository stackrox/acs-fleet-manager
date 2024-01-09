// Package leader provides a simple leader election mechanism for a pod
package serviceregistration

import (
	"context"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	activeLabel = "fleet-manager-active"
)

// leaderWorker uses kubernetes leader election to maintain a "fleetmanager-active":"true|false" label on the pod.
// this is used to route some requests to the leader pod.
// The default openshift Route will target all pods, but there is another Route that will target only the leader pod,
// (selector = fleetmanager-active: true) and will be used for requests that should only be handled by the leader.
// It also stops/starts the workers when the leader status changes.
type leaderWorker struct {
	namespaceName, podName string
	client                 kubernetes.Interface
	isLeader               atomic.Bool
	workers                []workers.Worker
}

// newWorker creates a new leaderWorker
func newWorker(namespaceName, podName string, client kubernetes.Interface, workers []workers.Worker) *leaderWorker {
	return &leaderWorker{
		namespaceName: namespaceName,
		podName:       podName,
		client:        client,
		workers:       workers,
	}
}

// run will run the leader election loop
func (l *leaderWorker) run(ctx context.Context) error {

	glog.V(1).Infoln("[serviceregistration] starting leader election")

	// the lock config
	lock := resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "fleet-manager-active-bd7c2840",
			Namespace: l.namespaceName,
		},
		Client: l.client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: l.podName,
		},
	}

	// create the leader election
	// the leaseDuration, renewDeadline and retryPeriod use the default recommended values
	leaderElection, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          &lock,
		LeaseDuration: time.Second * 15,
		RenewDeadline: time.Second * 10,
		RetryPeriod:   time.Second * 2,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				glog.Info("[serviceregistration] started leading")
				l.isLeader.Store(true)
				l.updateWithRetry(ctx)
				for _, worker := range l.workers {
					if !worker.IsRunning() {
						glog.V(1).Infoln(fmt.Sprintf("[serviceregistration] starting worker %q with id %q", worker.GetWorkerType(), worker.GetID()))
						worker.Start()
					} else {
						glog.V(1).Infoln(fmt.Sprintf("[serviceregistration] worker %q with id %q already running", worker.GetWorkerType(), worker.GetID()))
					}
				}
			},
			OnStoppedLeading: func() {
				glog.Info("[serviceregistration] stopped leading")
				l.isLeader.Store(false)
				l.updateWithRetry(ctx)
				for _, worker := range l.workers {
					if worker.IsRunning() {
						glog.V(1).Infoln(fmt.Sprintf("[serviceregistration] stopping worker %q with id %q", worker.GetWorkerType(), worker.GetID()))
						worker.Stop()
					} else {
						glog.V(1).Infoln(fmt.Sprintf("[serviceregistration] worker %q with id %q already stopped", worker.GetWorkerType(), worker.GetID()))
					}
				}
			},
		},
	})

	if err != nil {
		glog.Errorf("[serviceregistration] error creating leader election: %v", err)
		return errors.Wrap(err, "failed to create leader election")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				leaderElection.Run(ctx)
			}
		}
	}()

	return nil
}

const maxSleepMillis = 1000
const baseSleepMillis = 10

func (l *leaderWorker) updateWithRetry(ctx context.Context) {
	var attempt int
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := l.update(ctx); err == nil {
				return
			}
			// sleep min(1000, 10 * 2 ^ attempt milliseconds)
			sleepDuration := math.Min(baseSleepMillis*math.Pow(2, float64(attempt)), maxSleepMillis)
			time.Sleep(time.Duration(sleepDuration) * time.Millisecond)
			attempt++
		}
	}
}

// update will update the pod labels to reflect the current leader status
func (l *leaderWorker) update(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		isActive := strconv.FormatBool(l.isLeader.Load())

		// get the pod
		pod, err := l.client.CoreV1().Pods(l.namespaceName).Get(ctx, l.podName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("[serviceregistration] error getting pod: %v", err)
			return errors.Wrap(err, "failed to get pod")
		}

		// skip if the label is already set and has the correct value
		if pod.Labels != nil {
			value, ok := pod.Labels[activeLabel]
			if ok && value == isActive {
				glog.V(1).Infof("[serviceregistration] pod %s/%s already has the correct label %s=%s", l.namespaceName, l.podName, activeLabel, isActive)
				return nil
			}
		}

		glog.Infof("[serviceregistration] updating pod %s/%s labels: %s=%s", l.namespaceName, l.podName, activeLabel, isActive)
		pod.Labels[activeLabel] = isActive
		_, err = l.client.CoreV1().Pods(l.namespaceName).Update(ctx, pod, metav1.UpdateOptions{})
		if err != nil {
			glog.Errorf("[serviceregistration] error updating pod labels: %v", err)
			return errors.Wrap(err, "failed to patch pod")
		}
		return nil
	}
}
