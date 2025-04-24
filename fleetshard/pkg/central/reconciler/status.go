package reconciler

import (
	"strings"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
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
type StatusesCount struct {
	readyCentrals int
	totalCentrals int
}

// IncrementCurrent increments the counter for a given status
func (c *StatusesCount) IncrementCurrent(status private.DataPlaneCentralStatus) {
	c.IncrementRemote(deriveStatusKey(status))
}

// IncrementRemote increments the counter for a given key from the reported central
func (c *StatusesCount) IncrementRemote(status string) {
	c.totalCentrals++
	if status == centralConstants.CentralRequestStatusReady.String() {
		c.readyCentrals++
	}
}

// SubmitMetric sets corresponding prometheus metric of total central instances
func (c *StatusesCount) SubmitMetric() {
	fleetshardmetrics.MetricsInstance().SetTotalCentrals(c.totalCentrals)
	fleetshardmetrics.MetricsInstance().SetReadyCentrals(c.readyCentrals)
}

func deriveStatusKey(status private.DataPlaneCentralStatus) string {
	if status.Conditions == nil || len(status.Conditions) != 1 {
		return "error"
	}
	condition := status.Conditions[0]
	if condition.Type != "Ready" {
		return "error"
	}
	if condition.Status == "True" {
		return "ready"
	}
	return strings.ToLower(condition.Reason)
}
