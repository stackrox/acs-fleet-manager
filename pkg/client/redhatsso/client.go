package redhatsso

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

const (
	// access token duration before expiration
	tokenLifeDuration    = 5 * time.Minute
	cacheCleanupInterval = 299 * time.Second
)

// SSOClient ...
//go:generate moq -out client_moq.go . SSOClient
type SSOClient interface {
	GetToken() (string, error)
	GetConfig() *iam.IAMConfig
	GetRealmConfig() *iam.IAMRealmConfig
	GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error)
	GetServiceAccount(accessToken string, clientID string) (*serviceaccountsclient.ServiceAccountData, bool, error)
	CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	DeleteServiceAccount(accessToken string, clientID string) error
	UpdateServiceAccount(accessToken string, clientID string, name string, description string) (serviceaccountsclient.ServiceAccountData, error)
	RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error)
}

// NewSSOClient ...
func NewSSOClient(config *iam.IAMConfig, realmConfig *iam.IAMRealmConfig) SSOClient {
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
		},
		cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
	}
}

var _ SSOClient = &rhSSOClient{}

type rhSSOClient struct {
	config        *iam.IAMConfig
	realmConfig   *iam.IAMRealmConfig
	configuration *serviceaccountsclient.Configuration
	cache         *cache.Cache
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	Scope            string `json:"scope"`
}

func (c *rhSSOClient) getConfiguration(accessToken string) *serviceaccountsclient.Configuration {
	c.configuration = &serviceaccountsclient.Configuration{
		DefaultHeader: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", accessToken),
			"Content-Type":  "application/json",
		},
		UserAgent: "OpenAPI-Generator/1.0.0/go",
		Debug:     false,
		Servers: serviceaccountsclient.ServerConfigurations{
			{
				URL: c.realmConfig.BaseURL + c.realmConfig.APIEndpointURI,
			},
		},
	}

	return c.configuration
}

func (c *rhSSOClient) getCachedToken(tokenKey string) (string, error) {
	cachedToken, isCached := c.cache.Get(tokenKey)
	ct, _ := cachedToken.(string)
	if isCached {
		return ct, nil
	}
	return "", errors.Errorf("failed to retrieve cached token")
}

// GetToken ...
func (c *rhSSOClient) GetToken() (string, error) {
	cachedTokenKey := fmt.Sprintf("%s%s", c.realmConfig.Realm, c.realmConfig.ClientID)
	cachedToken, _ := c.getCachedToken(cachedTokenKey)

	if cachedToken != "" && !shared.IsJWTTokenExpired(cachedToken) {
		return cachedToken, nil
	}

	client := &http.Client{}
	parameters := url.Values{}
	parameters.Set("grant_type", "client_credentials")
	parameters.Set("client_id", c.realmConfig.ClientID)
	parameters.Set("client_secret", c.realmConfig.ClientSecret)
	req, err := http.NewRequest("POST", c.realmConfig.TokenEndpointURI, strings.NewReader(parameters.Encode()))
	if err != nil {
		return "", fmt.Errorf("getting token: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(parameters.Encode())))

	resp, err := client.Do(req)

	if err != nil {
		return "", fmt.Errorf("getting token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting token [%d]", resp.StatusCode)
	}

	token, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("getting token: %w", err)
	}

	var tokenData tokenResponse
	err = json.Unmarshal(token, &tokenData)
	if err != nil {
		return "", fmt.Errorf("getting token: %w", err)
	}
	c.cache.Set(cachedTokenKey, tokenData.AccessToken, cacheCleanupInterval)

	return tokenData.AccessToken, nil
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
func (c *rhSSOClient) GetServiceAccounts(accessToken string, first int, max int) ([]serviceaccountsclient.ServiceAccountData, error) {
	serviceAccounts, resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.GetServiceAccounts(context.Background()).
		Max(int32(max)).
		First(int32(first)).
		Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return nil, fmt.Errorf("getting service accounts: %w", err)
	}
	return serviceAccounts, nil
}

// GetServiceAccount ...
func (c *rhSSOClient) GetServiceAccount(accessToken string, clientID string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
	serviceAccount, resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.GetServiceAccount(context.Background(), clientID).
		Execute()

	defer shared.CloseResponseBody(resp)
	if resp != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, false, nil
		}
	}

	if err != nil {
		return nil, false, fmt.Errorf("getting service accounts: %w", err)
	}
	return &serviceAccount, err == nil, nil
}

// CreateServiceAccount ...
func (c *rhSSOClient) CreateServiceAccount(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	serviceAccount, resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.CreateServiceAccount(context.Background()).
		ServiceAccountCreateRequestData(
			serviceaccountsclient.ServiceAccountCreateRequestData{
				Name:        name,
				Description: description,
			}).Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return serviceAccount, fmt.Errorf("creating service account: %w", err)
	}
	return serviceAccount, nil
}

// DeleteServiceAccount ...
func (c *rhSSOClient) DeleteServiceAccount(accessToken string, clientID string) error {
	resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.DeleteServiceAccount(context.Background(), clientID).
		Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return fmt.Errorf("deleting service account: %w", err)
	}
	return nil
}

// UpdateServiceAccount ...
func (c *rhSSOClient) UpdateServiceAccount(accessToken string, clientID string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
	data, resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
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
func (c *rhSSOClient) RegenerateClientSecret(accessToken string, id string) (serviceaccountsclient.ServiceAccountData, error) {
	data, resp, err := serviceaccountsclient.NewAPIClient(c.getConfiguration(accessToken)).
		ServiceAccountsApi.
		ResetServiceAccountSecret(context.Background(), id).
		Execute()

	defer shared.CloseResponseBody(resp)

	if err != nil {
		return data, fmt.Errorf("regenerating client secret: %w", err)
	}
	return data, nil
}
