// Package dynamicclients ...
package dynamicclients

import (
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
)

// NewDynamicClientsAPI returns new instance of dynamic clients sso.redhat.com API client.
func NewDynamicClientsAPI(realmConfig *iam.IAMRealmConfig) *api.AcsTenantsApiService {
	// We count that the token being returned will contain api.iam.acs scope by default.
	// TODO(ROX-13408): switch to explicitly requesting api.iam.clients scope once it's available.
	httpClient := redhatsso.NewSSOAuthHTTPClient(realmConfig)
	configuration := &api.Configuration{
		BasePath:  realmConfig.BaseURL + realmConfig.APIEndpointURI,
		UserAgent: "RHACS-Fleet-Manager/1.0",
		Debug:     false,
		Servers: []api.ServerConfiguration{
			{
				Url: realmConfig.BaseURL + realmConfig.APIEndpointURI,
			},
		},
		HTTPClient: httpClient,
	}
	return api.NewAPIClient(configuration).AcsTenantsApi
}
