package dbapi

import "github.com/stackrox/acs-fleet-manager/pkg/api"

type DataPlaneClusterStatus struct {
	Conditions                        []DataPlaneClusterStatusCondition
	AvailableDinosaurOperatorVersions []api.DinosaurOperatorVersion
}

type DataPlaneClusterStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

type DataPlaneClusterConfigObservability struct {
	AccessToken string
	Channel     string
	Repository  string
	Tag         string
}

type DataPlaneClusterConfig struct {
	Observability DataPlaneClusterConfigObservability
}
