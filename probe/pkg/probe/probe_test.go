package probe

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	centralPkg "github.com/stackrox/acs-fleet-manager/probe/pkg/central"
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
	ProbeUsername:      "service-account-client",
}

var centralSpec = centralPkg.Spec{
	CloudProvider: "aws",
	Region:        "us-east-1",
}

func TestCreateCentral(t *testing.T) {
	tt := []struct {
		testName    string
		wantErr     bool
		errType     *error
		serviceMock *centralPkg.ServiceMock
	}{
		{
			testName: "create central happy path",
			wantErr:  false,
			serviceMock: &centralPkg.ServiceMock{
				CreateFunc: func(ctx context.Context, name string, spec centralPkg.Spec) (public.CentralRequest, error) {
					central := public.CentralRequest{
						Name:         name,
						Status:       constants.CentralRequestStatusAccepted.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil
				},
				GetFunc: func(ctx context.Context, id string) (public.CentralRequest, error) {
					central := public.CentralRequest{
						Id:           id,
						Status:       constants.CentralRequestStatusReady.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil
				},
			},
		},
		{
			testName: "create central fails on internal server error",
			wantErr:  true,
			serviceMock: &centralPkg.ServiceMock{
				CreateFunc: func(ctx context.Context, name string, spec centralPkg.Spec) (public.CentralRequest, error) {
					central := public.CentralRequest{Name: name}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return central, err
				},
			},
		},
		{
			testName: "central not ready on internal server error",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				CreateFunc: func(ctx context.Context, name string, spec centralPkg.Spec) (public.CentralRequest, error) {
					central := public.CentralRequest{
						Id:           "id-42",
						Name:         name,
						Status:       constants.CentralRequestStatusAccepted.String(),
						InstanceType: types.STANDARD.String(),
					}
					return central, nil
				},
				GetFunc: func(ctx context.Context, id string) (public.CentralRequest, error) {
					central := public.CentralRequest{
						Id: id,
					}
					err := errors.Errorf("%d", http.StatusInternalServerError)
					return central, err
				},
			},
		},
		{
			testName: "create central times out",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				CreateFunc: func(ctx context.Context, name string, spec centralPkg.Spec) (public.CentralRequest, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return public.CentralRequest{}, ctx.Err()
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.serviceMock, centralSpec)
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
		testName    string
		wantErr     bool
		errType     *error
		central     *public.CentralRequest
		serviceMock *centralPkg.ServiceMock
	}{
		{
			testName: "verify central happy path",
			wantErr:  false,
			central: &public.CentralRequest{
				Status:       constants.CentralRequestStatusReady.String(),
				InstanceType: types.STANDARD.String(),
			},
			serviceMock: &centralPkg.ServiceMock{
				PingFunc: func(ctx context.Context, url string) error {
					return nil
				},
			},
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
			serviceMock: &centralPkg.ServiceMock{
				PingFunc: func(ctx context.Context, url string) error {
					return fmt.Errorf("%d", http.StatusNotFound)
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.serviceMock, centralSpec)
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
		testName    string
		wantErr     bool
		errType     *error
		serviceMock *centralPkg.ServiceMock
	}{
		{
			testName: "delete central happy path",
			wantErr:  false,
			serviceMock: &centralPkg.ServiceMock{
				DeleteFunc: func(ctx context.Context, id string) error {
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (public.CentralRequest, error) {
					name := "delete central happy path"
					numGetCentralByIDCalls[name]++
					if numGetCentralByIDCalls[name] == 1 {
						return public.CentralRequest{
							Id:     "id-42",
							Name:   "probe-42",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil
					}
					return public.CentralRequest{}, centralPkg.ErrNotFound
				},
			},
		},
		{
			testName: "delete central fails on internal server error",
			wantErr:  true,
			serviceMock: &centralPkg.ServiceMock{
				DeleteFunc: func(ctx context.Context, id string) error {
					return errors.Errorf("%d", http.StatusInternalServerError)
				},
			},
		},
		{
			testName: "central not deprovision on internal server error",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				DeleteFunc: func(ctx context.Context, id string) error {
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (public.CentralRequest, error) {
					return public.CentralRequest{}, errors.Errorf("%d", http.StatusInternalServerError)
				},
			},
		},
		{
			testName: "central not deleted if no 404 response",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				DeleteFunc: func(ctx context.Context, id string) error {
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (public.CentralRequest, error) {
					name := "central not deleted if no 404 response"
					numGetCentralByIDCalls[name]++
					if numGetCentralByIDCalls[name] == 1 {
						return public.CentralRequest{
							Id:     "id-42",
							Name:   "probe-42",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil
					}

					return public.CentralRequest{
						Id:     "id-42",
						Name:   "probe-42",
						Status: constants.CentralRequestStatusDeleting.String(),
					}, nil
				},
			},
		},
		{
			testName: "delete central times out",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				DeleteFunc: func(ctx context.Context, id string) error {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return ctx.Err()
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.serviceMock, centralSpec)
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
		serviceMock     *centralPkg.ServiceMock
	}{
		{
			testName:        "clean up happy path",
			wantErr:         false,
			numDeleteCalled: 1,
			serviceMock: &centralPkg.ServiceMock{
				ListFunc: func(ctx context.Context, spec centralPkg.Spec) ([]public.CentralRequest, error) {
					name := "clean up happy path"
					numGetCentralByIDCalls[name]++
					var items []public.CentralRequest
					request := public.CentralRequest{
						Id:    "id-42",
						Name:  "probe-42",
						Owner: "service-account-client",
					}
					if numGetCentralByIDCalls[name] == 1 {
						request.Status = constants.CentralRequestStatusReady.String()
						items = append(items, request)
					}
					if numGetCentralByIDCalls[name] == 2 {
						request.Status = constants.CentralRequestStatusDeprovision.String()
						items = append(items, request)
					}
					return items, nil
				},
				DeleteFunc: func(ctx context.Context, id string) error {
					return nil
				},
			},
		},
		{
			testName: "nothing to clean up",
			wantErr:  false,
			serviceMock: &centralPkg.ServiceMock{
				ListFunc: func(ctx context.Context, spec centralPkg.Spec) ([]public.CentralRequest, error) {
					return []public.CentralRequest{
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
					}, nil
				},
			},
		},
		{
			testName: "clean up fails on internal server error",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				ListFunc: func(ctx context.Context, spec centralPkg.Spec) ([]public.CentralRequest, error) {
					return []public.CentralRequest{}, errors.Errorf("%d", http.StatusInternalServerError)
				},
			},
		},
		{
			testName: "clean up central times out",
			wantErr:  true,
			errType:  &context.DeadlineExceeded,
			serviceMock: &centralPkg.ServiceMock{
				ListFunc: func(ctx context.Context, spec centralPkg.Spec) ([]public.CentralRequest, error) {
					concurrency.WaitWithTimeout(ctx, 2*testConfig.ProbeRunTimeout)
					return []public.CentralRequest{}, ctx.Err()
				},
			},
		},
		{
			testName:        "clean up orphan",
			wantErr:         false,
			numDeleteCalled: 1,
			serviceMock: &centralPkg.ServiceMock{
				ListFunc: func(ctx context.Context, spec centralPkg.Spec) ([]public.CentralRequest, error) {
					name := "clean up orphan"
					numGetCentralByIDCalls[name]++
					items := []public.CentralRequest{
						{
							Id:        "id-42",
							Name:      "not-probe-42",
							Owner:     "service-account-client",
							CreatedAt: time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC),
							Status:    constants.CentralRequestStatusReady.String(),
						},
						{
							Id:        "id-43",
							Name:      "not-probe-43",
							Owner:     "service-account-client",
							CreatedAt: time.Now(),
							Status:    constants.CentralRequestStatusReady.String(),
						},
					}
					if numGetCentralByIDCalls[name] == 2 {
						items[0].Status = constants.CentralRequestStatusDeprovision.String()
					}
					if numGetCentralByIDCalls[name] == 3 {
						items = items[1:]
					}

					return items, nil
				},
				DeleteFunc: func(ctx context.Context, id string) error {
					return nil
				},
				GetFunc: func(ctx context.Context, id string) (public.CentralRequest, error) {
					name := "clean up orphan"
					numGetCentralByIDCalls[name]++
					if numGetCentralByIDCalls[name] == 1 {
						return public.CentralRequest{
							Id:     "id-42",
							Name:   "not-probe-42",
							Owner:  "service-account-client",
							Status: constants.CentralRequestStatusDeprovision.String(),
						}, nil
					} else if numGetCentralByIDCalls[name] == 2 {
						return public.CentralRequest{
							Id:     "id-43",
							Name:   "not-probe-43",
							Owner:  "service-account-client",
							Status: constants.CentralRequestStatusReady.String(),
						}, nil
					}

					return public.CentralRequest{}, errors.Errorf("%d", http.StatusNotFound)
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			probe := New(testConfig, tc.serviceMock, centralSpec)
			ctx, cancel := context.WithTimeout(context.TODO(), testConfig.ProbeRunTimeout)
			defer cancel()

			err := probe.cleanup(ctx)

			if tc.wantErr {
				assert.Error(t, err, "expected an error during probe run")
				if tc.errType != nil {
					assert.ErrorIs(t, err, *tc.errType)
				}
			} else {
				assert.NoError(t, err, "failed to delete central")
				assert.Equal(t, tc.numDeleteCalled, len(tc.serviceMock.DeleteCalls()))
			}
		})
	}
}
