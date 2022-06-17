package dbapi

import (
	"strings"
)

type DataPlaneCentralStatus struct {
	CentralClusterId string
	Conditions       []DataPlaneCentralStatusCondition
	// Going to ignore the rest of fields (like capacity and versions) for now, until when they are needed
	Routes                 []DataPlaneCentralRouteRequest
	CentralVersion         string
	CentralOperatorVersion string
}

type DataPlaneCentralStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

type DataPlaneCentralRoute struct {
	Domain string
	Router string
}

type DataPlaneCentralRouteRequest struct {
	Name   string
	Prefix string
	Router string
}

func (d *DataPlaneCentralStatus) GetReadyCondition() (DataPlaneCentralStatusCondition, bool) {
	for _, c := range d.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
			return c, true
		}
	}
	return DataPlaneCentralStatusCondition{}, false
}
