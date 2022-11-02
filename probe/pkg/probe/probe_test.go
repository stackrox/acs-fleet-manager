package probe

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/httpclient"
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

func makeHTTPResponse(statusCode int) *http.Response {
	response := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{},
		StatusCode: statusCode,
	}
	return response
}

func TestCreateCentral(t *testing.T) {
	tt := []struct {
		testName     string
		wantErr      bool
		errContains  string
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
			testName:    "create central fails on internal server error",
			wantErr:     true,
			errContains: "creation of central instance failed",
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return central, nil, err
				},
			},
		},
		{
			testName:    "central not ready on internal server error",
			wantErr:     true,
			errContains: "central instance id-42 did not reach ready state",
			mockFMClient: &fleetmanager.PublicClientMock{
				CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{
						Id:           "id-42",
						Name:         "probe-pod-42",
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
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe, err := New(testConfig, tc.mockFMClient, nil)
			require.NoError(t, err, "failed to create probe")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			central, err := probe.createCentral(ctx)

			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errContains, "expected an error during probe run")
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
		errContains    string
		central        *public.CentralRequest
		mockFMClient   *fleetmanager.PublicClientMock
		mockHTTPClient *http.Client
	}{
		{
			testName: "verify central happy path",
			wantErr:  false,
			central: &public.CentralRequest{
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			},
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{
						Status:       constants.CentralRequestStatusReady.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil, nil
				},
			},
			mockHTTPClient: httpclient.NewMockClient(func(req *http.Request) *http.Response {
				return makeHTTPResponse(http.StatusOK)
			}),
		},
		{
			testName:    "verify central fails if not standard instance",
			wantErr:     true,
			errContains: "central has wrong instance type: expected standard, got eval",
			central: &public.CentralRequest{
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.EVAL.String(),
			},
		},
		{
			testName:    "verify central fails if central UI not reachable",
			wantErr:     true,
			errContains: "could not reach central UI URL of instance id-42",
			central: &public.CentralRequest{
				Id:           "id-42",
				Name:         "probe-pod-42",
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			},
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
					central := public.CentralRequest{
						Status:       constants.CentralRequestStatusReady.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil, nil
				},
			},
			mockHTTPClient: httpclient.NewMockClient(func(req *http.Request) *http.Response {
				return makeHTTPResponse(http.StatusNotFound)
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe, err := New(testConfig, tc.mockFMClient, tc.mockHTTPClient)
			require.NoError(t, err, "failed to create probe")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err = probe.verifyCentral(ctx, tc.central)

			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errContains, "expected an error during probe run")
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
		errContains  string
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
							Name:   "probe-pod-42",
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
			testName:    "delete central fails on internal server error",
			wantErr:     true,
			errContains: "deletion of central instance id-42 failed",
			mockFMClient: &fleetmanager.PublicClientMock{
				DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return nil, err
				},
			},
		},
		{
			testName:    "central not deprovision on internal server error",
			wantErr:     true,
			errContains: "central instance id-42 did not reach deprovision state",
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
			testName:    "central not deleted if no 404 response",
			wantErr:     true,
			errContains: "central instance id-42 could not be deleted",
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
							Name:   "probe-pod-42",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil, nil
					}

					return public.CentralRequest{
						Id:     "id-42",
						Name:   "probe-pod-42",
						Status: constants.CentralRequestStatusDeleting.String(),
					}, nil, nil
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe, err := New(testConfig, tc.mockFMClient, nil)
			require.NoError(t, err, "failed to create probe")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			central := &public.CentralRequest{
				Id:           "id-42",
				Name:         "probe-pod-42",
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			}
			err = probe.deleteCentral(ctx, central)

			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errContains, "expected an error during probe run")
			} else {
				assert.NoError(t, err, "failed to delete central")
			}
		})
	}
}

func TestCleanUp(t *testing.T) {
	numGetCentralByIDCalls := make(map[string]int)

	tt := []struct {
		testName     string
		wantErr      bool
		errContains  string
		mockFMClient *fleetmanager.PublicClientMock
	}{
		{
			testName: "clean up happy path",
			wantErr:  false,
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					centralItems := []public.CentralRequest{
						{
							Id:   "id-42",
							Name: "probe-pod-42",
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
							Name:   "probe-pod-42",
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
					return public.CentralRequestList{}, nil, nil
				},
			},
		},
		{
			testName:    "clean up fails on internal server error",
			wantErr:     true,
			errContains: "could not list central",
			mockFMClient: &fleetmanager.PublicClientMock{
				GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
					centralList := public.CentralRequestList{}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return centralList, nil, err
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe, err := New(testConfig, tc.mockFMClient, nil)
			require.NoError(t, err, "failed to create probe")
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			cleanupDone := concurrency.NewSignal()
			err = probe.CleanUp(ctx, cleanupDone)
			require.True(t, cleanupDone.IsDone())

			if tc.wantErr {
				assert.ErrorContains(t, err, tc.errContains, "expected an error during probe run")
			} else {
				assert.NoError(t, err, "failed to delete central")
			}
		})
	}
}
