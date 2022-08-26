package dbapi

import "github.com/stackrox/acs-fleet-manager/pkg/api"

// DataPlaneClusterStatus ...
type DataPlaneClusterStatus struct {
	Conditions                        []DataPlaneClusterStatusCondition
	AvailableDinosaurOperatorVersions []api.CentralOperatorVersion
}

// DataPlaneClusterStatusCondition ...
type DataPlaneClusterStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}
