package services

import (
	"context"
	"testing"

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

	ctx := context.Background()
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
			auth := &TelemetryAuthMock{
				getUserFromContextFunc: func(ctx context.Context) (string, error) {
					return "user", nil
				},
			}
			tel := &telemetry.TelemeterMock{
				TrackFunc: func(event string, props map[string]any, opts ...telemeter.Option) {},
			}
			config := &telemetry.TelemetryConfigMock{
				EnabledFunc: func() bool {
					return true
				},
				TelemeterFunc: func() telemeter.Telemeter {
					return tel
				},
			}

			telemetry := NewTelemetry(auth, config)
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
