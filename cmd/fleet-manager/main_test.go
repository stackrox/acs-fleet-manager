package main

import (
	"os"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/shared/testutils"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"

	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

func TestInjections(t *testing.T) {
	RegisterTestingT(t)

	env, err := environments.New(environments.DevelopmentEnv,
		dinosaur.ConfigProviders(),
	)

	// Puts non-empty central IdP client secret value so config validation does not fail.
	file := testutils.CreateNonEmptyFile(t)
	defer os.Remove(file.Name())

	// Run env.CreateServices() via command to make use of --central-idp-client-secret-file flag.
	command := createServicesCommand(env)
	Expect(err).To(BeNil())
	err = env.AddFlags(command.Flags())
	Expect(err).To(BeNil())
	command.SetArgs([]string{"--central-idp-client-secret-file", file.Name()})
	err = command.Execute()
	Expect(err).To(BeNil())

	var bootList []environments.BootService
	env.MustResolve(&bootList)
	Expect(len(bootList)).To(Equal(4))

	_, ok := bootList[0].(*server.APIServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[1].(*server.MetricsServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[2].(*server.HealthCheckServer)
	Expect(ok).To(Equal(true))
	_, ok = bootList[3].(*workers.LeaderElectionManager)
	Expect(ok).To(Equal(true))

	var workerList []workers.Worker
	env.MustResolve(&workerList)
	Expect(workerList).To(HaveLen(10))
}

func createServicesCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createServices",
		Short: "Create Services",
		Long:  "Create Service",
		Run: func(cmd *cobra.Command, args []string) {
			err := env.CreateServices()
			if err != nil {
				glog.Fatalf("Unable to initialize environment: %s", err.Error())
			}
		},
	}
	return cmd
}
