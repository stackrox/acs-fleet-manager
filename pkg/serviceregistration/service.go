package serviceregistration

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
	"k8s.io/client-go/kubernetes/fake"
	"os"

	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Service is the service for service registration
type Service struct {
	ctx     context.Context
	cancel  context.CancelFunc
	workers []workers.Worker
}

// NewService returns a new Service.
func NewService(workers []workers.Worker) *Service {
	return &Service{
		workers: workers,
	}
}

// Start implements Service.Start
func (s *Service) Start() {
	if !features.LeaderElectionEnabled.Enabled() {
		return
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	var client kubernetes.Interface
	var namespaceName, podName string
	if e2e, _ := os.LookupEnv("E2E"); e2e == "true" {
		client = fake.NewSimpleClientset()
		namespaceName = "e2e-namespace"
		podName = "e2e-test-pod"
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		var ok bool
		client = kubernetes.NewForConfigOrDie(config)
		namespaceName, ok = os.LookupEnv("NAMESPACE_NAME")
		if !ok {
			panic("NAMESPACE_NAME not set")
		}
		podName, ok = os.LookupEnv("POD_NAME")
		if !ok {
			panic("POD_NAME not set")
		}
	}

	l, err := newWorker(s.ctx, namespaceName, podName, client, s.workers)
	if err != nil {
		panic(err)
	}
	go l.run(s.ctx)
}

// Stop implements Service.Stop
func (s *Service) Stop() {
	if !features.LeaderElectionEnabled.Enabled() {
		return
	}
	s.cancel()
}
