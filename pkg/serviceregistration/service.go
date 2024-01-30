package serviceregistration

import (
	"context"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// Service is the service for service registration
type Service struct {
	ctx          context.Context
	cancel       func()
	workers      []workers.Worker
	serverConfig *server.ServerConfig
}

// NewService returns a new Service.
func NewService(workers []workers.Worker, serverConfig *server.ServerConfig) *Service {
	return &Service{
		workers:      workers,
		serverConfig: serverConfig,
	}
}

// Start implements Service.Start
func (s *Service) Start() {
	glog.Info("starting service registration")
	if !s.serverConfig.EnableLeaderElection {
		glog.Info("leader election disabled")
		s.cancel = func() {
			stopWorkers(s.workers)
		}
		startWorkers(s.workers)
		return
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	l := newWorker(s.workers)
	if err := l.run(s.ctx); err != nil {
		glog.Errorf("error running leader election: %v", err)
		panic(err)
	}
}

// Stop implements Service.Stop
func (s *Service) Stop() {
	s.cancel()
}
