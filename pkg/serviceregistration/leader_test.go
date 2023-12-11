package serviceregistration

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLeader(t *testing.T) {
	c := fake.NewSimpleClientset(
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    map[string]string{},
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
				Labels:    map[string]string{},
			},
		},
	)
	worker1 := &leaderWorker{
		namespaceName: "default",
		podName:       "pod1",
		notify:        make(chan struct{}),
		client:        c,
		isLeader:      atomic.Bool{},
	}

	worker2 := &leaderWorker{
		namespaceName: "default",
		podName:       "pod2",
		notify:        make(chan struct{}),
		client:        c,
		isLeader:      atomic.Bool{},
	}

	worker1Ctx, worker1Cancel := context.WithCancel(context.Background())
	worker2Ctx := context.Background()

	// start worker 1
	worker1.run(worker1Ctx)

	// worker 1 should become leader
	require.Eventually(t, worker1.isLeader.Load, 1*time.Second, 100*time.Millisecond, "should be a leader")

	// start worker 2
	worker2.run(worker2Ctx)

	// stop worker 1
	worker1Cancel()

	// worker 2 should become leader
	require.Eventually(t, worker2.isLeader.Load, 20*time.Second, 100*time.Millisecond, "should be a leader")

}

func TestLeader_update(t *testing.T) {
	ctx := context.Background()

	c := fake.NewSimpleClientset(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels:    map[string]string{},
		},
	})

	w := &leaderWorker{
		namespaceName: "default",
		podName:       "pod1",
		notify:        make(chan struct{}),
		client:        c,
		isLeader:      atomic.Bool{},
	}

	w.isLeader.Store(true)
	w.update(ctx)

	// check that the pod has the active label = true
	pod, err := c.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "true", pod.Labels[activeLabel])

	w.isLeader.Store(false)
	w.update(ctx)

	// check that the pod has the active label = false
	pod, err = c.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "false", pod.Labels[activeLabel])
}
