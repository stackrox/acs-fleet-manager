package dbapi

// DataPlaneClusterStatus ...
type DataPlaneClusterStatus struct {
	Conditions []DataPlaneClusterStatusCondition
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
