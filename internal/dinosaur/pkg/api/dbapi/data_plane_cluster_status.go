package dbapi

// AddonInstallation represents the actual information about addons installed on the cluster
type AddonInstallation struct {
	Name                string
	Version             string
	SourceImage         string
	PackageImage        string
	ParametersSHA256Sum string
}

// DataPlaneClusterStatus is the actual state reported from the data plane cluster
type DataPlaneClusterStatus struct {
	Addons []AddonInstallation
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
