package redhatsso

import (
	"context"
	"fmt"
	"net/http"

	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// SSOClient ...
//
//go:generate moq -out client_moq.go . SSOClient
type SSOClient interface {
	GetConfig() *iam.IAMConfig
	GetRealmConfig() *iam.IAMRealmConfig
	GetServiceAccounts(first int, max int) ([]serviceaccountsclient.ServiceAccountData, error)
	GetServiceAccount(clientID string) (*serviceaccountsclient.ServiceAccountData, bool, error)
	CreateServiceAccount(name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	DeleteServiceAccount(clientID string) error
	UpdateServiceAccount(clientID string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	RegenerateClientSecret(id string) (serviceaccountsclient.ServiceAccountData, error)
}

// NewSSOClient ...
func NewSSOClient(config *iam.IAMConfig, realmConfig *iam.IAMRealmConfig) SSOClient {
	httpClient := NewSSOAuthHTTPClient(realmConfig, "api.iam.service_accounts")
	return &rhSSOClient{
		config:      config,
		realmConfig: realmConfig,
		configuration: &serviceaccountsclient.Configuration{
			UserAgent: "OpenAPI-Generator/1.0.0/go",
			Debug:     false,
			Servers: serviceaccountsclient.ServerConfigurations{
				{
					URL: realmConfig.BaseURL + realmConfig.APIEndpointURI,
				},
			},
			HTTPClient: httpClient,
		},
	}
}

var _ SSOClient = &rhSSOClient{}

type rhSSOClient struct {
	config        *iam.IAMConfig
	realmConfig   *iam.IAMRealmConfig
	configuration *serviceaccountsclient.Configuration
}

// GetConfig ...
func (c *rhSSOClient) GetConfig() *iam.IAMConfig {
	return c.config
}

// GetRealmConfig ...
func (c *rhSSOClient) GetRealmConfig() *iam.IAMRealmConfig {
	return c.realmConfig
}

// GetServiceAccounts ...
func (c *rhSSOClient) GetServiceAccounts(first int, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
	serviceAccounts, resp, err := serviceaccountsclient.NewAPIClient(c.configuration).
		ServiceAccountsApi.GetServiceAccounts(context.Background()).
		Max(int32(max)).
		First(int32(first)).
		Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return serviceAccounts, fmt.Errorf("getting service accounts: %w", err)
	}
	return serviceAccounts, nil
}

// GetServiceAccount ...
func (c *rhSSOClient) GetServiceAccount(clientID string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
	serviceAccount, resp, err := serviceaccountsclient.NewAPIClient(c.configuration).
		ServiceAccountsApi.GetServiceAccount(context.Background(), clientID).
		Execute()

	defer shared.CloseResponseBody(resp)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}

	if err != nil {
		return &serviceAccount, false, fmt.Errorf("getting service accounts: %w", err)
	}
	return &serviceAccount, true, nil
}

// CreateServiceAccount ...
func (c *rhSSOClient) CreateServiceAccount(name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	serviceAccount, resp, err := serviceaccountsclient.NewAPIClient(c.configuration).
		ServiceAccountsApi.CreateServiceAccount(context.Background()).
		ServiceAccountCreateRequestData(
			serviceaccountsclient.ServiceAccountCreateRequestData{
				Name:        name,
				Description: &description,
			}).Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return serviceAccount, fmt.Errorf("creating service account: %w", err)
	}
	return serviceAccount, nil
}

// DeleteServiceAccount ...
func (c *rhSSOClient) DeleteServiceAccount(clientID string) error {
	resp, err := serviceaccountsclient.NewAPIClient(c.configuration).
		ServiceAccountsApi.DeleteServiceAccount(context.Background(), clientID).
		Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return fmt.Errorf("deleting service account: %w", err)
	}
	return nil
}

// UpdateServiceAccount ...
func (c *rhSSOClient) UpdateServiceAccount(clientID string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	data, resp, err := serviceaccountsclient.NewAPIClient(c.configuration).
		ServiceAccountsApi.UpdateServiceAccount(context.Background(), clientID).
		ServiceAccountRequestData(serviceaccountsclient.ServiceAccountRequestData{
			Name:        &name,
			Description: &description,
		}).Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return data, fmt.Errorf("updating service accounts: %w", err)
	}
	return data, nil
}

// RegenerateClientSecret ...
func (c *rhSSOClient) RegenerateClientSecret(id string) (serviceaccountsclient.ServiceAccountData, error) {
	data, resp, err := serviceaccountsclient.NewAPIClient(c.configuration).
		ServiceAccountsApi.
		ResetServiceAccountSecret(context.Background(), id).
		Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return data, fmt.Errorf("regenerating client secret: %w", err)
	}
	return data, nil
}
