package dbapi

// FleetshardAddonStatus represents the actual information about the fleetshard addon installed on the cluster
type FleetshardAddonStatus struct {
	Version             string
	SourceImage         string
	PackageImage        string
	ParametersSHA256Sum string
}

// DataPlaneClusterStatus is the actual state reported from the data plane cluster
type DataPlaneClusterStatus struct {
	FleetshardAddonStatus FleetshardAddonStatus
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
