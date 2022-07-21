package api

// CloudProvider ...
type CloudProvider struct {
	Kind        string `json:"kind"`
	Id          string `json:"id"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
}

// CloudProviderList ...
type CloudProviderList []*CloudProvider
