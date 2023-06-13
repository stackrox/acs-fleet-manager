package centrals

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultVersionCommand(t *testing.T) {

	defaultTestVersion := "testregistry.com/testversion:123"
	defaultExpectedOutput := fmt.Sprintf("Central Default Version set to: %s\n", defaultTestVersion)

	tt := []struct {
		name            string
		args            []string
		buildMockClient func() *fleetmanager.Client
		expectedOut     func() (string, error)
		expectedErr     func() (string, error)
	}{
		{
			name: "should output succesfull response",
			args: []string{defaultTestVersion},
			buildMockClient: func() *fleetmanager.Client {
				fmMock := fleetmanager.NewClientMock()
				fmMock.AdminAPIMock.SetCentralDefaultVersionFunc = func(ctx context.Context, centralDefaultVersionPayload private.CentralDefaultVersion) (*http.Response, error) {
					return nil, nil
				}

				return fmMock.Client()
			},
			expectedOut: func() (string, error) {
				return defaultExpectedOutput, nil
			},
			expectedErr: func() (string, error) {
				return "", nil
			},
		},
		{
			name: "should output api error message",
			args: []string{defaultTestVersion},
			buildMockClient: func() *fleetmanager.Client {
				fmMock := fleetmanager.NewClientMock()
				fmMock.AdminAPIMock.SetCentralDefaultVersionFunc = func(ctx context.Context, centralDefaultVersionPayload private.CentralDefaultVersion) (*http.Response, error) {
					return nil, errors.New("test error")
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
			cmd := NewAdminCentralsSetDefaultVersionCommand(tc.buildMockClient())
			outBuf := bytes.Buffer{}
			errBuf := bytes.Buffer{}

			cmd.SetOut(&outBuf)
			cmd.SetErr(&errBuf)
			cmd.SetArgs(tc.args)
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
