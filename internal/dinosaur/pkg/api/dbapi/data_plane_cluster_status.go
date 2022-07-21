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

// DataPlaneClusterConfigObservability ...
type DataPlaneClusterConfigObservability struct {
	AccessToken string
	Channel     string
	Repository  string
	Tag         string
}

// DataPlaneClusterConfig ...
type DataPlaneClusterConfig struct {
	Observability DataPlaneClusterConfigObservability
}
