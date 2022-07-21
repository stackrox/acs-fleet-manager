package api

// CloudRegion ...
type CloudRegion struct {
	Kind                   string   `json:"kind"`
	Id                     string   `json:"id"`
	DisplayName            string   `json:"display_name"`
	CloudProvider          string   `json:"cloud_provider"`
	Enabled                bool     `json:"enabled"`
	SupportedInstanceTypes []string `json:"supported_instance_types"`
}

// CloudRegionList ...
type CloudRegionList *[]CloudRegion
