package dbapi

import (
	"strings"
)

// DataPlaneCentralStatus ...
type DataPlaneCentralStatus struct {
	CentralClusterId string
	Conditions       []DataPlaneCentralStatusCondition
	// Going to ignore the rest of fields (like capacity and versions) for now, until when they are needed
	Routes                 []DataPlaneCentralRouteRequest
	CentralVersion         string
	CentralOperatorVersion string
}

// DataPlaneCentralStatusCondition ...
type DataPlaneCentralStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

// DataPlaneCentralRoute ...
type DataPlaneCentralRoute struct {
	Domain string
	Router string
}

// DataPlaneCentralRouteRequest ...
type DataPlaneCentralRouteRequest struct {
	Name   string
	Prefix string
	Router string
}

// GetReadyCondition ...
func (d *DataPlaneCentralStatus) GetReadyCondition() (DataPlaneCentralStatusCondition, bool) {
	for _, c := range d.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
			return c, true
		}
	}
	return DataPlaneCentralStatusCondition{}, false
}
