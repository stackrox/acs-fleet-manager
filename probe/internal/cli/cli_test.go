package cli

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/runtime"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = &config.Config{
	ProbeRunTimeout: 10 * time.Millisecond,
	ProbeName:       "pod",
	ProbeNamePrefix: "probe",
	RHSSOClientID:   "client",
}

func TestCLIInterrupt(t *testing.T) {
	fleetManagerClient := &fleetmanager.PublicClientMock{
		CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
			process, err := os.FindProcess(os.Getpid())
			require.NoError(t, err, "could not find current process ID")
			process.Signal(os.Interrupt)

			concurrency.Wait(ctx)
			return public.CentralRequest{}, nil, ctx.Err()
		},
		GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
			return public.CentralRequestList{}, nil, nil
		},
	}
	runtime, err := runtime.New(testConfig, fleetManagerClient, nil)
	require.NoError(t, err, "failed to create runtime")
	cli := &CLI{runtime: runtime}
	cmd := cli.Command()
	cmd.SetArgs([]string{"run"})

	err = cmd.Execute()

	assert.ErrorIs(t, err, errInterruptSignal, "did not receive interrupt signal")
}
