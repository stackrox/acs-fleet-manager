package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
)

var testConfig = config.Config{
	ProbePollPeriod:    10 * time.Millisecond,
	ProbeRunTimeout:    100 * time.Millisecond,
	ProbeRunWaitPeriod: 10 * time.Millisecond,
	ProbeName:          "probe",
	RHSSOClientID:      "client",
}

func TestRunSingle(t *testing.T) {
	tt := []struct {
		testName    string
		serviceMock *central.ServiceMock
	}{
		{
			testName: "deadline exceeded on time out in Create",
			serviceMock: &central.ServiceMock{
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
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return public.CentralRequest{}, ctx.Err()
				},
			},
		},
		{
			testName: "deadline exceeded on time out in List",
			serviceMock: &central.ServiceMock{
				ListSpecsFunc: func(ctx context.Context) ([]central.Spec, error) {
					return []central.Spec{
						{
							Region:        "us-east-1",
							CloudProvider: "aws",
						},
					}, nil
				},
				ListFunc: func(ctx context.Context, spec central.Spec) ([]public.CentralRequest, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return []public.CentralRequest{}, ctx.Err()
				},
				CreateFunc: func(ctx context.Context, name string, spec central.Spec) (public.CentralRequest, error) {
					return public.CentralRequest{}, nil
				},
			},
		},
		{
			testName: "deadline exceeded on time out in ListSpecs",
			serviceMock: &central.ServiceMock{
				ListSpecsFunc: func(ctx context.Context) ([]central.Spec, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return []central.Spec{}, ctx.Err()
				},
				ListFunc: func(ctx context.Context, spec central.Spec) ([]public.CentralRequest, error) {
					return []public.CentralRequest{}, nil
				},
				CreateFunc: func(ctx context.Context, name string, spec central.Spec) (public.CentralRequest, error) {
					return public.CentralRequest{}, nil
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {

			runtime := New(testConfig, tc.serviceMock)
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err := runtime.RunSingle(ctx)

			assert.ErrorIs(t, err, context.DeadlineExceeded)
		})
	}
}
