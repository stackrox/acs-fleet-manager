package main

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

func TestInjections(t *testing.T) {
	RegisterTestingT(t)

	env, err := environments.New(environments.DevelopmentEnv,
		dinosaur.ConfigProviders(),
	)
	Expect(err).To(BeNil())
	err = env.CreateServices()
	Expect(err).To(BeNil())

	var bootList []environments.BootService
	env.MustResolve(&bootList)
	Expect(len(bootList)).To(Equal(4))

	_, ok := bootList[0].(*server.ApiServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[1].(*server.MetricsServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[2].(*server.HealthCheckServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[3].(*workers.LeaderElectionManager)
	Expect(ok).To(Equal(true))

	var workerList []workers.Worker
	env.MustResolve(&workerList)
	Expect(workerList).To(HaveLen(8))

}
