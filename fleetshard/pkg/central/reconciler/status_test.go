package reconciler

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/require"
)

func TestStatusesCount(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []private.DataPlaneCentralStatus
		wantInvalid    int32
		wantReady      int32
		wantInstalling int32
		wantDeleted    int32
	}{
		{
			name: "should contain invalid when status is empty",
			statuses: []private.DataPlaneCentralStatus{
				{},
			},
			wantInvalid: 1,
		},
		{
			name: "should contain invalid when conditions is nil",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: nil,
				},
			},
			wantInvalid: 1,
		},
		{
			name: "should contain invalid when conditions is empty",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: []private.DataPlaneCentralStatusConditions{},
				},
			},
			wantInvalid: 1,
		},
		{
			name: "should contain invalid when conditions length more than one",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: []private.DataPlaneCentralStatusConditions{
						{Type: "Ready", Status: "True"},
						{Type: "Ready", Status: "True"},
					},
				},
			},
			wantInvalid: 1,
		},
		{
			name: "should contain invalid when conditions type is not ready",
			statuses: []private.DataPlaneCentralStatus{
				{
					Conditions: []private.DataPlaneCentralStatusConditions{
						{Type: "NotReady", Status: "True"},
					},
				},
			},
			wantInvalid: 1,
		},
		{
			name: "should contain ready when conditions status is ready",
			statuses: []private.DataPlaneCentralStatus{
				*readyStatus(),
			},
			wantReady: 1,
		},
		{
			name: "should contain installing when condition is ready and reason is installing",
			statuses: []private.DataPlaneCentralStatus{
				*installingStatus(),
			},
			wantInstalling: 1,
		},
		{
			name: "should contain installing when condition is ready and reason is installing",
			statuses: []private.DataPlaneCentralStatus{
				*deletedStatus(),
			},
			wantDeleted: 1,
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
			wantReady:      3,
			wantInstalling: 2,
			wantDeleted:    1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			counter := StatusesCount{}
			for _, status := range test.statuses {
				counter.Increment(status)
			}
			require.Equal(t, test.wantInvalid, counter["Invalid"])
			require.Equal(t, test.wantReady, counter["Ready"])
			require.Equal(t, test.wantInstalling, counter["Installing"])
			require.Equal(t, test.wantDeleted, counter["Deleted"])
		})
	}
}
