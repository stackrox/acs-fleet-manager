package runtime

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = &config.Config{
	RuntimePollPeriod:    1 * time.Second,
	RuntimePollTimeout:   10 * time.Second,
	RuntimeRunTimeout:    10 * time.Second,
	RuntimeRunWaitPeriod: 1 * time.Second,
}

func TestRuntimeProbeCentral(t *testing.T) {
	tt := []struct {
		testName     string
		responseFunc func(*fleetmanager.MockClient)
		wantErr      bool
		errContains  string
	}{
		{
			testName: "testProbeCentralHappyPath",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr: false,
		},
		{
			testName: "testProbeCentralCreateStatusServerError",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseStatusServerError)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr:     true,
			errContains: "Creation request to Central instance was not accepted",
		},
		{
			testName: "testProbeCentralCreateCentralError",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseError)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr:     true,
			errContains: "Creation of Central instance failed: Failed response",
		},
		{
			testName: "testProbeCentralPollingDoesNotRespondOK",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusInternalServerError}
			},
			wantErr:     true,
			errContains: "Central UI  did not respond with status OK. Got:",
		},
		{
			testName: "testProbeCentralGetCentralStatusServerErrorIsRetried",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseStatusServerError)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr: false,
		},
		{
			testName: "testProbeCentralGetCentralErrorIsRetried",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseError)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr: false,
		},
		{
			testName: "testProbeCentralDeleteCentralError",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseError)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr:     true,
			errContains: "Deletion of Central instance  failed: Failed response",
		},
		{
			testName: "testProbeCentralDeleteCentralStatusServerError",
			responseFunc: func(client *fleetmanager.MockClient) {
				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseStatusServerError)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr:     true,
			errContains: "Deletion request to Central instance  was not accepted",
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			client, err := fleetmanager.NewMock()
			require.NoError(t, err, "Failed to create fleet manager client.")
			tc.responseFunc(client)
			runtime, err := New(testConfig, client)
			require.NoError(t, err, "Failed to create runtime.")

			ctx := context.Background()
			err = runtime.probeCentral(ctx)

			if tc.wantErr {
				require.ErrorContains(t, err, tc.errContains, "Expected an error during probe run.")
			} else {
				require.NoError(t, err, "Failed to run probe.")
			}
		})
	}
}

func TestRuntimeTimeout(t *testing.T) {
	tt := []struct {
		testName     string
		responseFunc func(*fleetmanager.MockClient)
		wantErr      bool
	}{
		{
			testName: "testProbeCentralRunTimesOut",
			responseFunc: func(client *fleetmanager.MockClient) {
				testConfig.RuntimeRunTimeout = 0

				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr: true,
		},
		{
			testName: "testProbeCentralPollingTimesOut",
			responseFunc: func(client *fleetmanager.MockClient) {
				testConfig.RuntimePollTimeout = 0

				client.AddCreateCentralResponse(fleetmanager.CreateCentralResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseReady)
				client.AddDeleteCentralByIDResponse(fleetmanager.DeleteCentralByIDResponseAccepted)
				client.AddGetCentralByIDResponse(fleetmanager.GetCentralByIDResponseDeprovision)

				httpclient.HTTPClient = &httpclient.MockClient{StatusCode: http.StatusOK}
			},
			wantErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			client, err := fleetmanager.NewMock()
			require.NoError(t, err, "Failed to create fleet manager client.")
			tc.responseFunc(client)
			runtime, err := New(testConfig, client)
			require.NoError(t, err, "Failed to create runtime.")

			isErr := make(chan bool, 1)
			sigs := make(chan os.Signal, 1)
			runtime.Run(isErr, sigs, false)

			if tc.wantErr {
				assert.True(t, <-isErr, "Expected an error during probe run.")
			} else {
				assert.False(t, <-isErr, "Did not expect an error during probe run.")
			}
		})
	}
}
