package leader

import (
	"context"
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

	ctx1, cancel1 := context.WithCancel(context.Background())
	l1 := &worker{
		namespaceName: "default",
		podName:       "pod1",
		notify:        make(chan struct{}),
		client:        c,
		isLeader:      false,
	}

	// start pod1
	l1.run(ctx1)

	// pod1 will become leader
	require.Eventually(t, func() bool {
		return l1.isLeader
	}, 1*time.Second, 100*time.Millisecond, "should be a leader")

	ctx2 := context.Background()
	l2 := &worker{
		namespaceName: "default",
		podName:       "pod2",
		notify:        make(chan struct{}),
		client:        c,
		isLeader:      false,
	}
	// start pod2
	l2.run(ctx2)

	// stop pod1
	cancel1()

	// pod2 will become leader
	require.Eventually(t, func() bool {
		return l2.isLeader
	}, 20*time.Second, 100*time.Millisecond, "should be a leader")

}

func TestLeader_update(t *testing.T) {
	c := fake.NewSimpleClientset(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels:    map[string]string{},
		},
	})
	ctx := context.Background()
	l1 := &worker{
		namespaceName: "default",
		podName:       "pod1",
		notify:        make(chan struct{}),
		client:        c,
		isLeader:      true,
	}

	l1.update(ctx)

	pod, err := c.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "true", pod.Labels[activeLabel])

	l1.isLeader = false
	l1.update(ctx)
	pod, err = c.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "false", pod.Labels[activeLabel])
}
