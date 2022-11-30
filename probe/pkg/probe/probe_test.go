package probe

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
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
	ProbeUsername:       "service-account-client",
}

func makeHTTPResponse(statusCode int) *http.Response {
	response := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{},
		StatusCode: statusCode,
	}
	return response
}

func newHTTPClientMock(fn httputil.RoundTripperFunc) *http.Client {
	return &http.Client{
		Transport: httputil.RoundTripperFunc(fn),
	}
}

func TestGenerateCentralName(t *testing.T) {
	probe := New(testConfig, nil, nil)
	for i := 0; i < 10; i++ {
		centralName, err := probe.generateCentralName()
		require.NoError(t, err)

		matchedName := handlers.ValidCentralNameRegexp.MatchString(centralName)
		matchedLength := len(centralName) < handlers.MaxCentralNameLength
		assert.True(t, matchedName)
		assert.True(t, matchedLength)
	}
}

func TestCreateCentral(t *testing.T) {
	tt := []struct {
		testName     string
		wantErr      bool
		errType      *error
		mockFMClient *fleetmanager.PublicClientMock
	}{
		{
			testName: "create central happy path",
			wantErr:  false,
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{
						Status:       constants.CentralRequestStatusAccepted.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil, nil
				},
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{
						Status:       constants.CentralRequestStatusReady.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil, nil
				},
			},
		},
		{
			testName: "create central fails on internal server error",
			wantErr:  true,
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return central, nil, err
				},
			},
		},
		{
			testName: "central not ready on internal server error",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{
						Id:           "id-42",
						Name:         "probe-42",
						Status:       constants.CentralRequestStatusAccepted.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil, nil
				},
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return central, nil, err
				},
			},
		},
		{
			testName: "create central times out",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return public.CentralRequest{}, nil, ctx.Err()
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.mockFMClient, nil)
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			central, err := probe.createCentral(ctx)

			if tc.wantErr {
				assert.Error(t, err, "expected an error during probe run")
				if tc.errType != nil {
					assert.ErrorIs(t, err, *tc.errType)
				}
			} else {
				require.NoError(t, err, "failed to create central")
				assert.Equal(t, constants.CentralRequestStatusReady.String(), central.Status, "central not ready")
			}
		})
	}
}

func TestVerifyCentral(t *testing.T) {
	tt := []struct {
		testName       string
		wantErr        bool
		errType        *error
		central        *public.CentralRequest
		mockHTTPClient *http.Client
	}{
		{
			testName: "verify central happy path",
			wantErr:  false,
			central: &public.CentralRequest{
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			},
			mockHTTPClient: newHTTPClientMock(func(req *http.Request) (*http.Response, error) {
				return makeHTTPResponse(http.StatusOK), nil
			}),
		},
		{
			testName: "verify central fails if not standard instance",
			wantErr:  true,
			central: &public.CentralRequest{
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.EVAL.String(),
			},
		},
		{
			testName: "verify central fails if central UI not reachable",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			central: &public.CentralRequest{
				Id:           "id-42",
				Name:         "probe-42",
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			},
			mockHTTPClient: newHTTPClientMock(func(req *http.Request) (*http.Response, error) {
				return makeHTTPResponse(http.StatusNotFound), nil
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, nil, tc.mockHTTPClient)
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err := probe.verifyCentral(ctx, tc.central)

			if tc.wantErr {
				assert.Error(t, err, "expected an error during probe run")
				if tc.errType != nil {
					assert.ErrorIs(t, err, *tc.errType)
				}
			} else {
				assert.NoError(t, err, "failed to verify central")
			}
		})
	}
}

func TestDeleteCentral(t *testing.T) {
	numGetCentralByIDCalls := make(map[string]int)

	tt := []struct {
		testName     string
		wantErr      bool
		errType      *error
		mockFMClient *fleetmanager.PublicClientMock
	}{
		{
			testName: "delete central happy path",
			wantErr:  false,
			mockFMClient: &fleetmanager.PublicClientMock{
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					return nil, nil
				},
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					name := "delete central happy path"
					numGetCentralByIDCalls[name]++
					if numGetCentralByIDCalls[name] == 1 {
						return public.CentralRequest{
							Id:     "id-42",
							Name:   "probe-42",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil, nil
					}

					central := public.CentralRequest{}
					response := makeHTTPResponse(http.StatusNotFound)
					err := errors.Errorf("%d", http.StatusNotFound)
					return central, response, err
				},
			},
		},
		{
			testName: "delete central fails on internal server error",
			wantErr:  true,
			mockFMClient: &fleetmanager.PublicClientMock{
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return nil, err
				},
			},
		},
		{
			testName: "central not deprovision on internal server error",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					return nil, nil
				},
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return central, nil, err
				},
			},
		},
		{
			testName: "central not deleted if no 404 response",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					return nil, nil
				},
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					name := "central not deleted if no 404 response"
					numGetCentralByIDCalls[name]++
					if numGetCentralByIDCalls[name] == 1 {
						return public.CentralRequest{
							Id:     "id-42",
							Name:   "probe-42",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil, nil
					}

					return public.CentralRequest{
						Id:     "id-42",
						Name:   "probe-42",
						Status: constants.CentralRequestStatusDeleting.String(),
					}, nil, nil
				},
			},
		},
		{
			testName: "delete central times out",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return nil, ctx.Err()
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.mockFMClient, nil)
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			central := &public.CentralRequest{
				Id:           "id-42",
				Name:         "probe-42",
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			}
			err := probe.deleteCentral(ctx, central)

			if tc.wantErr {
				assert.Error(t, err, "expected an error during probe run")
				if tc.errType != nil {
					assert.ErrorIs(t, err, *tc.errType)
				}
			} else {
				assert.NoError(t, err, "failed to delete central")
			}
		})
	}
}

func TestCleanUp(t *testing.T) {
	numGetCentralByIDCalls := make(map[string]int)

	tt := []struct {
		testName        string
		wantErr         bool
		errType         *error
		numDeleteCalled int
		mockFMClient    *fleetmanager.PublicClientMock
	}{
		{
			testName:        "clean up happy path",
			wantErr:         false,
			numDeleteCalled: 1,
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					centralItems := []public.CentralRequest{
						{
							Id:    "id-42",
							Name:  "probe-42",
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
					name := "clean up happy path"
					numGetCentralByIDCalls[name]++
					if numGetCentralByIDCalls[name] == 1 {
						return public.CentralRequest{
							Id:     "id-42",
							Name:   "probe-42",
							Owner:  "service-account-client",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil, nil
					}

					central := public.CentralRequest{}
					response := makeHTTPResponse(http.StatusNotFound)
					err := errors.Errorf("%d", http.StatusNotFound)
					return central, response, err
				},
			},
		},
		{
			testName: "nothing to clean up",
			wantErr:  false,
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					centralItems := []public.CentralRequest{
						{
							Id:    "id-42",
							Name:  "probe-42",
							Owner: "service-account-wrong-owner",
						},
						{
							Id:    "id-42",
							Name:  "wrong-name-42",
							Owner: "service-account-wrong-owner",
						},
					}
					centralList := public.CentralRequestList{Items: centralItems}
					return centralList, nil, nil
				},
			},
		},
		{
			testName: "clean up fails on internal server error",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					centralList := public.CentralRequestList{}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return centralList, nil, err
				},
			},
		},
		{
			testName: "clean up central times out",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeCleanUpTimeout)
					return public.CentralRequestList{}, nil, ctx.Err()
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.mockFMClient, nil)
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err := probe.CleanUp(ctx)

			if tc.wantErr {
				assert.Error(t, err, "expected an error during probe run")
				if tc.errType != nil {
					assert.ErrorIs(t, err, *tc.errType)
				}
			} else {
				assert.NoError(t, err, "failed to delete central")
				assert.Equal(t, tc.numDeleteCalled, len(tc.mockFMClient.DeleteCentralByIdCalls()))
			}
		})
	}
}
