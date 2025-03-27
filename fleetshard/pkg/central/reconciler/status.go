package reconciler

import (
	"strings"

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
type StatusesCount struct {
	readyCentrals int
	totalCentrals int
	errorCentrals int
}

// IncrementWithStatus increments the counter for a given status
func (c *StatusesCount) IncrementWithStatus(status private.DataPlaneCentralStatus) {
	c.Increment(deriveStatusKey(status))
}

// IncrementError increments the error counter
func (c *StatusesCount) IncrementError() {
	c.Increment("error")
}

// Increment increments the counter for a given key
func (c *StatusesCount) Increment(key string) {
	c.totalCentrals++
	if key == "ready" {
		c.readyCentrals++
	}
	if key == "error" {
		c.errorCentrals++
	}
}

// SubmitMetric sets corresponding prometheus metric of total central instances
func (c *StatusesCount) SubmitMetric() {
	fleetshardmetrics.MetricsInstance().SetTotalCentrals(c.totalCentrals)
	fleetshardmetrics.MetricsInstance().SetReadyCentrals(c.readyCentrals)
	fleetshardmetrics.MetricsInstance().AddCentralReconcilationErrors(c.errorCentrals)
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
