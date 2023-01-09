package fleetmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	fleetManagerErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stretchr/testify/assert"
)

const clusterID = "1234567890abcdef1234567890abcdef" // pragma: allowlist secret

func TestFormatAPIError(t *testing.T) {
	type testCase struct {
		err                error
		expectErrorMessage string
	}

	cases := map[string]testCase{
		"should format nil error to nil string": {
			err:                nil,
			expectErrorMessage: "nil",
		},
		"should format API error with message": {
			err: emulatePrivateAPIError(400, "400 Bad Request",
				fleetManagerErrors.BadRequest("Cluster agent with ID '%s' not found", clusterID)),
			expectErrorMessage: "400 Bad Request (RHACS-MGMT-21: Cluster agent with ID '1234567890abcdef1234567890abcdef' not found)", // pragma: allowlist secret
		},
		"should format API error without message": {
			err:                emulatePrivateAPIError(418, "418 I'm a teapot", nil),
			expectErrorMessage: "418 I'm a teapot",
		},
		"should format an arbitrary error": {
			err:                errors.New("test error"),
			expectErrorMessage: "test error",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			actual := FormatAPIError(c.err)
			assert.Equal(t, c.expectErrorMessage, actual)
		})
	}
}

func emulatePrivateAPIError(statusCode int, status string, serviceError *fleetManagerErrors.ServiceError) error {
	config := private.NewConfiguration()
	config.HTTPClient = newHTTPClientMock(func(req *http.Request) (*http.Response, error) {
		header := http.Header{}
		header.Add("Content-Type", "application/json")

		bodyBuf := &bytes.Buffer{}
		operationID := logger.GetOperationID(req.Context())

		if serviceError != nil {
			err := json.NewEncoder(bodyBuf).Encode(serviceError.AsOpenapiError(operationID, req.RequestURI))
			if err != nil {
				return nil, err
			}
		}

		return &http.Response{
			Body:       io.NopCloser(bodyBuf),
			Header:     header,
			StatusCode: statusCode,
			Status:     status,
		}, nil
	})
	client := private.NewAPIClient(config)
	_, _, err := client.AgentClustersApi.GetDataPlaneClusterAgentConfig(context.TODO(), clusterID)
	return err
}

func newHTTPClientMock(fn httputil.RoundTripperFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}
