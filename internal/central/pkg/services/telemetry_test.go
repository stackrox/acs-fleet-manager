package services

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryTrackRequests(t *testing.T) {
	type testCase struct {
		name       string
		isAdmin    bool
		requestErr error
		trackFunc  func(t *Telemetry, tt testCase)
	}
	defaultToken := &jwt.Token{
		Claims: jwt.MapClaims{
			"user_id": "user",
		},
	}
	ctx := authentication.ContextWithToken(context.Background(), defaultToken)
	tenantID := "tenant-id"
	createFunc := func(t *Telemetry, tt testCase) {
		t.trackCreationRequested(ctx, tenantID, tt.isAdmin, tt.requestErr)
	}
	deleteFunc := func(t *Telemetry, tt testCase) {
		t.TrackDeletionRequested(ctx, tenantID, tt.isAdmin, tt.requestErr)
	}

	tests := []testCase{
		{
			name:       "creation with admin API, no request error",
			isAdmin:    true,
			requestErr: nil,
			trackFunc:  createFunc,
		},
		{
			name:       "creation with public API, no request error",
			isAdmin:    false,
			requestErr: nil,
			trackFunc:  createFunc,
		},
		{
			name:       "creation with admin API, with request error",
			isAdmin:    true,
			requestErr: errors.New("request error"),
			trackFunc:  createFunc,
		},
		{
			name:       "creation with public API, with request error",
			isAdmin:    false,
			requestErr: errors.New("request error"),
			trackFunc:  createFunc,
		},
		{
			name:       "deletion with admin API, no request error",
			isAdmin:    true,
			requestErr: nil,
			trackFunc:  deleteFunc,
		},
		{
			name:       "deletion with public API, no request error",
			isAdmin:    false,
			requestErr: nil,
			trackFunc:  deleteFunc,
		},
		{
			name:       "deletion with admin API, with request error",
			isAdmin:    true,
			requestErr: errors.New("request error"),
			trackFunc:  deleteFunc,
		},
		{
			name:       "deletion with public API, with request error",
			isAdmin:    false,
			requestErr: errors.New("request error"),
			trackFunc:  deleteFunc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel := &TelemeterMock{
				TrackFunc: func(event string, props map[string]any, opts ...telemeter.Option) {},
			}
			telemetry := &Telemetry{
				telemeter: tel,
			}
			tt.trackFunc(telemetry, tt)

			calls := tel.TrackCalls()
			require.Len(t, calls, 1)
			assert.Equal(t, tenantID, calls[0].Props["Tenant ID"])
			assert.Equal(t, tt.isAdmin, calls[0].Props["Is Admin Request"])
			assert.Equal(t, tt.requestErr == nil, calls[0].Props["Success"])
			if tt.requestErr != nil {
				assert.Equal(t, tt.requestErr.Error(), calls[0].Props["Error"])
			}
		})
	}
}

func TestNilTelemeter(t *testing.T) {
	tel := NewTelemetry(&telemetry.TelemetryConfig{})
	assert.Nil(t, tel.telemeter, "telemeter should be nil")
	ctx := context.Background()
	centralRequest := buildCentralRequest(nil)
	// no nil pointer errors
	tel.RegisterTenant(ctx, centralRequest, false, nil)
	tel.UpdateTenantProperties(centralRequest)
	tel.TrackDeletionRequested(ctx, "tenant-id", false, nil)
	tel.Stop()
}
