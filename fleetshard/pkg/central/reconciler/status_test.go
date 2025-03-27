package reconciler

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/require"
)

func TestStatusesCount(t *testing.T) {
	tests := []struct {
		name      string
		statuses  []private.DataPlaneCentralStatus
		wantTotal int
		wantReady int
		wantError int
	}{
		{
			name: "should contain error when status is empty",
			statuses: []private.DataPlaneCentralStatus{
				{},
			},
			wantTotal: 1,
			wantError: 1,
		},
		{
			name: "should contain error when conditions is nil",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: nil,
				},
			},
			wantTotal: 1,
			wantError: 1,
		},
		{
			name: "should contain error when conditions is empty",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: []private.DataPlaneCentralStatusConditions{},
				},
			},
			wantTotal: 1,
			wantError: 1,
		},
		{
			name: "should contain error when conditions length more than one",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: []private.DataPlaneCentralStatusConditions{
						{Type: "Ready", Status: "True"},
						{Type: "Ready", Status: "True"},
					},
				},
			},
			wantError: 1,
			wantTotal: 1,
		},
		{
			name: "should contain error when conditions type is not ready",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: []private.DataPlaneCentralStatusConditions{
						{Type: "NotReady", Status: "True"},
					},
				},
			},
			wantTotal: 1,
			wantError: 1,
		},
		{
			name: "should contain ready when conditions status is ready",
			statuses: []private.DataPlaneCentralStatus{
				*readyStatus(),
			},
			wantReady: 1,
			wantTotal: 1,
		},
		{
			name: "should contain installing when condition is ready and reason is installing",
			statuses: []private.DataPlaneCentralStatus{
				*installingStatus(),
			},
			wantTotal: 1,
		},
		{
			name: "should contain installing when condition is ready and reason is installing",
			statuses: []private.DataPlaneCentralStatus{
				*deletedStatus(),
			},
			wantTotal: 1,
			wantReady: 0,
		},
		{
			name: "should combine multiple statuses",
			statuses: []private.DataPlaneCentralStatus{
				*readyStatus(),
				*readyStatus(),
				*readyStatus(),
				*installingStatus(),
				*installingStatus(),
				*deletedStatus(),
			},
			wantReady: 3,
			wantTotal: 6,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			counter := StatusesCount{}
			for _, status := range test.statuses {
				counter.IncrementWithStatus(status)
			}
			require.Equal(t, test.wantTotal, counter.totalCentrals)
			require.Equal(t, test.wantReady, counter.readyCentrals)
			require.Equal(t, test.wantError, counter.errorCentrals)
		})
	}
}
