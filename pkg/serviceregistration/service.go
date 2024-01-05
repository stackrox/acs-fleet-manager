package serviceregistration

import (
	"context"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
	"k8s.io/client-go/kubernetes/fake"
	"os"

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
	glog.Info("starting service registration")
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
	l := newWorker(namespaceName, podName, client, s.workers)
	if err := l.run(s.ctx); err != nil {
		glog.Errorf("error running leader election: %v", err)
		panic(err)
	}
}

// Stop implements Service.Stop
func (s *Service) Stop() {
	s.cancel()
}
