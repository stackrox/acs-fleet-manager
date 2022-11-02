package cli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = &config.Config{
	ProbeCleanUpTimeout: 100 * time.Millisecond,
	ProbeRunTimeout:     100 * time.Millisecond,
	ProbeName:           "pod",
	ProbeNamePrefix:     "probe",
	RHSSOClientID:       "client",
}

func TestCLIInterrupt(t *testing.T) {
	mockProbe := &probe.ProbeMock{
		CleanUpFunc: func(ctx context.Context, done concurrency.Signal) error {
			done.Signal()
			return nil
		},
		ExecuteFunc: func(ctx context.Context) error {
			process, err := os.FindProcess(os.Getpid())
			require.NoError(t, err, "could not find current process ID")
			process.Signal(os.Interrupt)

			concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
			return ctx.Err()
		},
	}
	runtime, err := runtime.New(testConfig, mockProbe)
	require.NoError(t, err, "failed to create runtime")
	cli := &CLI{runtime: runtime}
	cmd := cli.Command()
	cmd.SetArgs([]string{"run"})

	err = cmd.Execute()

	assert.ErrorIs(t, err, errInterruptSignal, "did not receive interrupt signal")
	assert.Equal(t, 1, len(mockProbe.CleanUpCalls()), "must clean up centrals")
}
