package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = &config.Config{
	ProbePollPeriod:     10 * time.Millisecond,
	ProbeCleanUpTimeout: 100 * time.Millisecond,
	ProbeRunTimeout:     100 * time.Millisecond,
	ProbeRunWaitPeriod:  10 * time.Millisecond,
	ProbeName:           "probe",
	RHSSOClientID:       "client",
}

func TestRunSingle(t *testing.T) {
	tt := []struct {
		testName  string
		mockProbe *probe.ProbeMock
	}{
		{
			testName: "deadline exceeded on time out in Execute",
			mockProbe: &probe.ProbeMock{
				CleanUpFunc: func(ctx context.Context) error {
					return nil
				},
				ExecuteFunc: func(ctx context.Context) error {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return ctx.Err()
				},
			},
		},
		{
			testName: "deadline exceeded on out in CleanUp",
			mockProbe: &probe.ProbeMock{
				CleanUpFunc: func(ctx context.Context) error {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeCleanUpTimeout)
					return ctx.Err()
				},
				ExecuteFunc: func(ctx context.Context) error {
					return nil
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			runtime, err := New(testConfig, tc.mockProbe)
			require.NoError(t, err, "failed to create runtime")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err = runtime.RunSingle(ctx)

			assert.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Equal(t, 1, len(tc.mockProbe.CleanUpCalls()), "must clean up centrals")
		})
	}
}

func TestCanceledContextStillCleansUp(t *testing.T) {
	tt := []struct {
		testName  string
		mockProbe *probe.ProbeMock
	}{
		{
			testName: "cancel main context before cleanup timeout",
			mockProbe: &probe.ProbeMock{
				CleanUpFunc: func(ctx context.Context) error {
					return ctx.Err()
				},
				ExecuteFunc: func(ctx context.Context) error {
					concurrency.WaitWithTimeout(ctx, 10*time.Millisecond)
					return ctx.Err()
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			runtime, err := New(testConfig, tc.mockProbe)
			require.NoError(t, err, "failed to create runtime")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			go func() {
				time.Sleep(5 * time.Millisecond)
				cancel()
			}()
			err = runtime.RunSingle(ctx)

			assert.NotContains(t, err.Error(), errCleanupFailed.Error())
			assert.Equal(t, 1, len(tc.mockProbe.CleanUpCalls()), "must clean up centrals")
		})
	}
}
