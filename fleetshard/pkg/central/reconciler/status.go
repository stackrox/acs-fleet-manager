package reconciler

import "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"

func readyStatus() *private.DataPlaneCentralStatus {
	return &private.DataPlaneCentralStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}
}

func deletedStatus() *private.DataPlaneCentralStatus {
	return &private.DataPlaneCentralStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "False",
				Reason: "Deleted",
			},
		},
	}
}

func installingStatus() *private.DataPlaneCentralStatus {
	return &private.DataPlaneCentralStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "False",
				Reason: "Installing",
			},
		},
	}
}
