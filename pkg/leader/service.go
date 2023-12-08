package leader

import (
	"context"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Service is the service for leader election.
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewService returns a new Service.
func NewService() *Service {
	return &Service{}
}

// Start implements Service.Start
func (s *Service) Start() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	client := kubernetes.NewForConfigOrDie(config)
	namespaceName, ok := os.LookupEnv("NAMESPACE_NAME")
	if !ok {
		panic("NAMESPACE_NAME not set")
	}
	podName, ok := os.LookupEnv("POD_NAME")
	if !ok {
		panic("POD_NAME not set")
	}
	l, err := newWorker(s.ctx, namespaceName, podName, client)
	if err != nil {
		panic(err)
	}
	go l.run(s.ctx)
}

// Stop implements Service.Stop
func (s *Service) Stop() {
	s.cancel()
}
