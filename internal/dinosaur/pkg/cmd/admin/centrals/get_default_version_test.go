package centrals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultVersionCommand(t *testing.T) {

	defaultVersionResponse := private.CentralDefaultVersion{
		Version: "testversion",
	}

	tt := []struct {
		name            string
		buildMockClient func() *fleetmanager.Client
		expectedOut     func() (string, error)
		expectedErr     func() (string, error)
	}{
		{
			name: "should output CentralDefaultVersion as json",
			buildMockClient: func() *fleetmanager.Client {
				fmMock := fleetmanager.NewClientMock()
				fmMock.AdminAPIMock.GetCentralDefaultVersionFunc = func(ctx context.Context) (private.CentralDefaultVersion, *http.Response, error) {
					return defaultVersionResponse, nil, nil
				}

				return fmMock.Client()
			},
			expectedOut: func() (string, error) {
				out, err := json.Marshal(&defaultVersionResponse)
				return string(out), err
			},
			expectedErr: func() (string, error) {
				return "", nil
			},
		},
		{
			name: "should output api error message",
			buildMockClient: func() *fleetmanager.Client {
				fmMock := fleetmanager.NewClientMock()
				fmMock.AdminAPIMock.GetCentralDefaultVersionFunc = func(ctx context.Context) (private.CentralDefaultVersion, *http.Response, error) {
					return private.CentralDefaultVersion{}, nil, errors.New("test error")
				}
				return fmMock.Client()
			},
			expectedOut: func() (string, error) {
				return "", nil
			},
			expectedErr: func() (string, error) {
				return "error calling fleet-manager API: test error\n", nil
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewAdminCentralsGetDefaultVersionCommand(tc.buildMockClient())
			outBuf := bytes.Buffer{}
			errBuf := bytes.Buffer{}

			cmd.SetOut(&outBuf)
			cmd.SetErr(&errBuf)

			cmd.Execute()
			outB, err := io.ReadAll(&outBuf)
			require.NoError(t, err, "error reading out buffer")
			errB, err := io.ReadAll(&errBuf)
			require.NoError(t, err, "error reading err buffer")

			expectedOut, err := tc.expectedOut()
			require.NoError(t, err, "error generating expected stdout output")
			require.Equal(t, expectedOut, string(outB))

			expectedErr, err := tc.expectedErr()
			require.NoError(t, err, "error generating expected stderr output")
			require.Equal(t, expectedErr, string(errB))
		})
	}
}
