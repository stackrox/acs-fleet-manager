package cli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/central"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = config.Config{
	ProbePollPeriod:    10 * time.Millisecond,
	ProbeRunTimeout:    100 * time.Millisecond,
	ProbeRunWaitPeriod: 10 * time.Millisecond,
	ProbeName:          "probe",
	RHSSOClientID:      "client",
}

func TestCLIInterrupt(t *testing.T) {
	serviceMock := &central.ServiceMock{
		ListSpecsFunc: func(ctx context.Context) ([]central.Spec, error) {
			return []central.Spec{
				{
					Region:        "us-east-1",
					CloudProvider: "aws",
				},
			}, nil
		},
		ListFunc: func(ctx context.Context, spec central.Spec) ([]public.CentralRequest, error) {
			return []public.CentralRequest{}, nil
		},
		CreateFunc: func(ctx context.Context, name string, spec central.Spec) (public.CentralRequest, error) {
			process, err := os.FindProcess(os.Getpid())
			require.NoError(t, err, "could not find current process ID")
			_ = process.Signal(os.Interrupt)
			concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
			return public.CentralRequest{}, ctx.Err()
		},
	}
	runtime := runtime.New(testConfig, serviceMock)
	cli := &CLI{runtime: runtime}
	cmd := cli.Command()
	cmd.SetArgs([]string{"run"})

	err := cmd.Execute()

	assert.ErrorIs(t, err, errInterruptSignal, "did not receive interrupt signal")
}
