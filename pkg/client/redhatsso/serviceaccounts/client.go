// Package serviceaccounts ...
package serviceaccounts

import (
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"

	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-core/app-services-sdk-go/serviceaccountmgmt/apiv1/client"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
)

// NewServiceAccountsAPI returns new instance of service accounts sso.redhat.com API client.
func NewServiceAccountsAPI(realmConfig *iam.IAMRealmConfig) serviceaccountsclient.ServiceAccountsApi {
	httpClient := redhatsso.NewSSOAuthHTTPClient(realmConfig, "api.iam.service_accounts")
	configuration := &serviceaccountsclient.Configuration{
		UserAgent: "RHACS-Fleet-Manager/1.0",
		Debug:     false,
		Servers: serviceaccountsclient.ServerConfigurations{
			{
				URL: realmConfig.BaseURL + realmConfig.APIEndpointURI,
			},
		},
		HTTPClient: httpClient,
	}
	return serviceaccountsclient.NewAPIClient(configuration).ServiceAccountsApi
}
