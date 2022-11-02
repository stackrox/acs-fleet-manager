package runtime

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = &config.Config{
	ProbePollPeriod:     10 * time.Millisecond,
	ProbeCleanUpTimeout: 100 * time.Millisecond,
	ProbeRunTimeout:     100 * time.Millisecond,
	ProbeRunWaitPeriod:  10 * time.Millisecond,
	ProbeName:           "pod",
	ProbeNamePrefix:     "probe",
	RHSSOClientID:       "client",
}

func TestRunSingle(t *testing.T) {
	tt := []struct {
		testName     string
		mockFMClient *fleetmanager.PublicClientMock
	}{
		{
			testName: "centrals are cleaned up on time out",
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return public.CentralRequest{}, nil, ctx.Err()
				},
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					centralItems := []public.CentralRequest{
						{
							Id:    "id-42",
							Name:  "probe-pod-42",
							Owner: "service-account-client",
						},
					}
					centralList := public.CentralRequestList{Items: centralItems}
					return centralList, nil, nil
				},
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					return nil, nil
				},
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					return public.CentralRequest{}, nil, nil
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			runtime, err := New(testConfig, tc.mockFMClient, nil)
			require.NoError(t, err, "failed to create runtime")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err = runtime.RunSingle(ctx)

			assert.ErrorContains(t, err, "probe run failed", "expected an error during probe run")
			assert.ErrorContains(t, err, "context deadline exceeded", "expected timeout error")
			assert.Equal(t, 1, len(tc.mockFMClient.GetCentralsCalls()), 1, "must retrieve central list for clean up")
			assert.Equal(t, 1, len(tc.mockFMClient.DeleteCentralByIdCalls()), 1, "must delete central for clean up")
			assert.Equal(t, "id-42", tc.mockFMClient.DeleteCentralByIdCalls()[0].ID, "deleted central ID did not match")
		})
	}
}
