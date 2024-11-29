package dbapi

// AddonInstallation represents the actual information about addons installed on the cluster
type AddonInstallation struct {
	ID                  string
	Version             string
	SourceImage         string
	PackageImage        string
	ParametersSHA256Sum string
}

// DataPlaneClusterStatus is the actual state reported from the data plane cluster
type DataPlaneClusterStatus struct {
	Addons []AddonInstallation
}

// DataPlaneClusterConfig ...
type DataPlaneClusterConfig struct {
}
