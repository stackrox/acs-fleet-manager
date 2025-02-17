package reconciler

import (
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

func readyStatus() *private.DataPlaneCentralStatus {
	return &private.DataPlaneCentralStatus{
		Conditions: []private.DataPlaneCentralStatusConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}
}

func deletedStatus() *private.DataPlaneCentralStatus {
	return &private.DataPlaneCentralStatus{
		Conditions: []private.DataPlaneCentralStatusConditions{
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
		Conditions: []private.DataPlaneCentralStatusConditions{
			{
				Type:   "Ready",
				Status: "False",
				Reason: "Installing",
			},
		},
	}
}

// StatusesCount is a container that holds counters grouped by statuses
type StatusesCount map[string]int32

// Increment increments the counter for a given status
func (c StatusesCount) Increment(status private.DataPlaneCentralStatus) {
	c[deriveStatusKey(status)]++
}

// SubmitMetric sets corresponding prometheus metric of total central instances
func (c StatusesCount) SubmitMetric() {
	for key, count := range c {
		fleetshardmetrics.MetricsInstance().SetTotalCentrals(float64(count), key)
	}
}

func deriveStatusKey(status private.DataPlaneCentralStatus) string {
	if status.Conditions == nil || len(status.Conditions) != 1 {
		return "Invalid"
	}
	condition := status.Conditions[0]
	if condition.Type != "Ready" {
		return "Invalid"
	}
	if condition.Status == "True" {
		return "Ready"
	}
	return condition.Reason
}
