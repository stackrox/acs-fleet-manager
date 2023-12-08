// Package leader provides a simple leader election mechanism for a pod
package leader

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/retry"
)

const (
	activeLabel = "fleet-manager-active"
)

// worker uses kubernetes leader election to maintain a "fleetmanager-active":"true|false" label on the pod.
// this is used to route some requests to the leader pod.
// The default openshift Route will target all pods, but there is another Route that will target only the leader pod,
// (selector = fleetmanager-active: true) and will be used for requests that should only be handled by the leader.
type worker struct {
	namespaceName, podName string
	client                 kubernetes.Interface
	isLeader               atomic.Bool
	notify                 chan struct{}
}

// newWorker creates a new worker
func newWorker(ctx context.Context, namespaceName, podName string, client kubernetes.Interface) (*worker, error) {
	l := &worker{
		namespaceName: namespaceName,
		podName:       podName,
		client:        client,
		notify:        make(chan struct{}),
	}

	err := l.run(ctx)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// run will run the leader election loop
func (l *worker) run(ctx context.Context) error {

	// the lock config
	lock := resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "fleet-manager-leader-election",
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
				glog.Info("[leader] started leading")
				l.isLeader.Store(true)
				l.notify <- struct{}{}
			},
			OnStoppedLeading: func() {
				glog.Info("[leader] stopped leading")
				l.isLeader.Store(false)
				l.notify <- struct{}{}
			},
		},
	})

	if err != nil {
		return errors.Wrap(err, "[leader] failed to create leader election")
	}

	go func() {
		// start the leader election. It will stop when the context is cancelled
		leaderElection.Run(ctx)
	}()

	go func() {
		// update every time we get a notification
		for {
			select {
			case <-ctx.Done():
				return
			case <-l.notify:
				l.update(ctx)
			}
		}
	}()

	return nil
}

// update will update the pod labels to reflect the current leader status
func (l *worker) update(ctx context.Context) {

	// retry forever
	retry.OnError(retry.DefaultRetry, retryForever, func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
			isActive := strconv.FormatBool(l.isLeader.Load())

			// get the pod
			pod, err := l.client.CoreV1().Pods(l.namespaceName).Get(ctx, l.podName, metav1.GetOptions{})
			if err != nil {
				glog.Errorf("[leader] error getting pod: %v", err)
				return errors.Wrap(err, "failed to get pod")
			}

			// skip if the label is already set and has the correct value
			if pod.Labels != nil {
				value, ok := pod.Labels[activeLabel]
				if ok && value == isActive {
					return nil
				}
			}

			glog.Infof("[leader] updating pod %s/%s labels: %s=%s", l.namespaceName, l.podName, activeLabel, isActive)
			patch := `{"metadata":{"labels":{"` + activeLabel + `":"` + isActive + `"}}}`

			_, err = l.client.CoreV1().Pods(l.namespaceName).Patch(ctx, l.podName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
			if err != nil {
				glog.Errorf("[leader] error updating pod labels: %v", err)
				return errors.Wrap(err, "failed to patch pod")
			}
			return nil
		}
	})
}

var retryForever = func(err error) bool { return true }
