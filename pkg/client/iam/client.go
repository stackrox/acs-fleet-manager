package iam

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Nerzal/gocloak/v11"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

const (
	// gocloak access token duration before expiration
	tokenLifeDuration    = 5 * time.Minute
	cacheCleanupInterval = 5 * time.Minute
	// OrgKey ...
	OrgKey = "rh-org-id"
	// UserKey ...
	UserKey = "rh-user-id"
)

var (
	protocol = "openid-connect"
	mapper   = "oidc-usermodel-attribute-mapper"
)

// IAMClient ...
//go:generate moq -out client_moq.go . IAMClient
type IAMClient interface {
	CreateClient(client gocloak.Client, accessToken string) (string, error)
	GetToken() (string, error)
	GetCachedToken(tokenKey string) (string, error)
	DeleteClient(internalClientID string, accessToken string) error
	GetClientSecret(internalClientID string, accessToken string) (string, error)
	GetClient(clientID string, accessToken string) (*gocloak.Client, error)
	IsClientExist(clientID string, accessToken string) (string, error)
	GetConfig() *IAMConfig
	GetRealmConfig() *IAMRealmConfig
	GetClientByID(id string, accessToken string) (*gocloak.Client, error)
	ClientConfig(client ClientRepresentation) gocloak.Client
	CreateProtocolMapperConfig(string) []gocloak.ProtocolMapperRepresentation
	GetClientServiceAccount(accessToken string, internalClient string) (*gocloak.User, error)
	UpdateServiceAccountUser(accessToken string, serviceAccountUser gocloak.User) error
	// GetClients returns IAM clients using the given method parameters. If max is less than 0, then returns all the clients.
	// If it is 0, then default to using the default max allowed service accounts configuration.
	GetClients(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error)
	IsSameOrg(client *gocloak.Client, orgID string) bool
	IsOwner(client *gocloak.Client, userID string) bool
	RegenerateClientSecret(accessToken string, id string) (*gocloak.CredentialRepresentation, error)
	GetRealmRole(accessToken string, roleName string) (*gocloak.Role, error)
	CreateRealmRole(accessToken string, roleName string) (*gocloak.Role, error)
	UserHasRealmRole(accessToken string, userID string, roleName string) (*gocloak.Role, error)
	AddRealmRoleToUser(accessToken string, userID string, role gocloak.Role) error
}

// ClientRepresentation ...
type ClientRepresentation struct {
	Name                         string
	ClientID                     string
	ServiceAccountsEnabled       bool
	Secret                       *string
	StandardFlowEnabled          bool
	Attributes                   map[string]string
	AuthorizationServicesEnabled bool
	ProtocolMappers              []gocloak.ProtocolMapperRepresentation
	Description                  string
	RedirectURIs                 *[]string
}

type iamClient struct {
	kcClient    gocloak.GoCloak
	ctx         context.Context
	config      *IAMConfig
	realmConfig *IAMRealmConfig
	cache       *cache.Cache
}

var _ IAMClient = &iamClient{}

// GoCloak an alias for gocloak.GoCloak
//go:generate moq -out gocloak_moq.go . GoCloak
type GoCloak = gocloak.GoCloak

// NewClient ...
func NewClient(config *IAMConfig, realmConfig *IAMRealmConfig) *iamClient {
	setTokenEndpoints(config, realmConfig)
	client := gocloak.NewClient(config.BaseURL)
	client.RestyClient().SetDebug(config.Debug)
	client.RestyClient().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: config.InsecureSkipVerify})
	return &iamClient{
		kcClient:    client,
		ctx:         context.Background(),
		config:      config,
		realmConfig: realmConfig,
		cache:       cache.New(tokenLifeDuration, cacheCleanupInterval),
	}
}

// ClientConfig ...
func (ic *iamClient) ClientConfig(client ClientRepresentation) gocloak.Client {
	publicClient := false
	directAccess := false
	return gocloak.Client{
		Name:                         &client.Name,
		ClientID:                     &client.ClientID,
		ServiceAccountsEnabled:       &client.ServiceAccountsEnabled,
		StandardFlowEnabled:          &client.StandardFlowEnabled,
		Attributes:                   &client.Attributes,
		AuthorizationServicesEnabled: &client.AuthorizationServicesEnabled,
		ProtocolMappers:              &client.ProtocolMappers,
		Description:                  &client.Description,
		RedirectURIs:                 client.RedirectURIs,
		Protocol:                     &protocol,
		PublicClient:                 &publicClient,
		DirectAccessGrantsEnabled:    &directAccess,
	}
}

// CreateProtocolMapperConfig ...
func (ic *iamClient) CreateProtocolMapperConfig(name string) []gocloak.ProtocolMapperRepresentation {
	protocolMapper := []gocloak.ProtocolMapperRepresentation{
		{
			Name:           &name,
			Protocol:       &protocol,
			ProtocolMapper: &mapper,
			Config: &map[string]string{
				"access.token.claim":   "true",
				"claim.name":           name,
				"id.token.claim":       "true",
				"jsonType.label":       "String",
				"user.attribute":       name,
				"userinfo.token.claim": "true",
			},
		},
	}
	return protocolMapper
}

func setTokenEndpoints(config *IAMConfig, realmConfig *IAMRealmConfig) {
	realmConfig.JwksEndpointURI = config.BaseURL + "/auth/realms/" + realmConfig.Realm + "/protocol/openid-connect/certs"
	realmConfig.TokenEndpointURI = config.BaseURL + "/auth/realms/" + realmConfig.Realm + "/protocol/openid-connect/token"
	realmConfig.ValidIssuerURI = config.BaseURL + "/auth/realms/" + realmConfig.Realm
}

// CreateClient ...
func (ic *iamClient) CreateClient(client gocloak.Client, accessToken string) (string, error) {
	internalClientID, err := ic.kcClient.CreateClient(ic.ctx, accessToken, ic.realmConfig.Realm, client)
	if err != nil {
		return "", fmt.Errorf("creating client: %w", err)
	}
	return internalClientID, nil
}

// GetClient ...
func (ic *iamClient) GetClient(clientID string, accessToken string) (*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		ClientID: &clientID,
	}
	clients, err := ic.kcClient.GetClients(ic.ctx, accessToken, ic.realmConfig.Realm, params)
	if err != nil {
		return nil, fmt.Errorf("getting clients: %w", err)
	}
	for _, client := range clients {
		if *client.ClientID == clientID {
			return client, nil
		}
	}
	return nil, nil
}

// GetToken ...
func (ic *iamClient) GetToken() (string, error) {
	options := gocloak.TokenOptions{
		ClientID:     &ic.realmConfig.ClientID,
		GrantType:    &ic.realmConfig.GrantType,
		ClientSecret: &ic.realmConfig.ClientSecret,
	}
	cachedTokenKey := fmt.Sprintf("%s%s", ic.realmConfig.ValidIssuerURI, ic.realmConfig.ClientID)
	cachedToken, _ := ic.GetCachedToken(cachedTokenKey)

	if cachedToken != "" && !shared.IsJWTTokenExpired(cachedToken) {
		return cachedToken, nil
	}
	tokenResp, err := ic.kcClient.GetToken(ic.ctx, ic.realmConfig.Realm, options)
	if err != nil {
		return "", errors.Wrap(err, "failed to get new token from gocloak with error")
	}

	ic.cache.Set(cachedTokenKey, tokenResp.AccessToken, cacheCleanupInterval)
	return tokenResp.AccessToken, nil
}

// GetCachedToken ...
func (ic *iamClient) GetCachedToken(tokenKey string) (string, error) {
	cachedToken, isCached := ic.cache.Get(tokenKey)
	ct, _ := cachedToken.(string)
	if isCached {
		return ct, nil
	}
	return "", errors.Errorf("failed to retrieve cached token")
}

// GetClientSecret ...
func (ic *iamClient) GetClientSecret(internalClientID string, accessToken string) (string, error) {
	resp, err := ic.kcClient.GetClientSecret(ic.ctx, accessToken, ic.realmConfig.Realm, internalClientID)
	if err != nil {
		return "", fmt.Errorf("getting client secret: %w", err)
	}
	if resp.Value == nil {
		return "", errors.Errorf("failed to retrieve credentials")
	}
	return *resp.Value, nil
}

// DeleteClient ...
func (ic *iamClient) DeleteClient(internalClientID string, accessToken string) error {
	err := ic.kcClient.DeleteClient(ic.ctx, accessToken, ic.realmConfig.Realm, internalClientID)
	if err != nil {
		return fmt.Errorf("deleting client: %w", err)
	}
	return nil
}

func (ic *iamClient) getClient(clientID string, accessToken string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		ClientID: &clientID,
	}
	client, err := ic.kcClient.GetClients(ic.ctx, accessToken, ic.realmConfig.Realm, params)
	if err != nil {
		return nil, fmt.Errorf("getting clients: %w", err)
	}
	return client, nil
}

// GetClientByID ...
func (ic *iamClient) GetClientByID(internalID string, accessToken string) (*gocloak.Client, error) {
	client, err := ic.kcClient.GetClient(ic.ctx, accessToken, ic.realmConfig.Realm, internalID)
	if err != nil {
		return nil, fmt.Errorf("getting client by ID: %w", err)
	}
	return client, nil
}

// GetConfig ...
func (ic *iamClient) GetConfig() *IAMConfig {
	return ic.config
}

// GetRealmConfig ...
func (ic *iamClient) GetRealmConfig() *IAMRealmConfig {
	return ic.realmConfig
}

// IsClientExist ...
func (ic *iamClient) IsClientExist(clientID string, accessToken string) (string, error) {
	if clientID == "" {
		return "", errors.New("clientId cannot be empty")
	}
	clients, err := ic.getClient(clientID, accessToken)
	if err != nil {
		return "", err
	}
	for _, client := range clients {
		if *client.ClientID == clientID {
			return *client.ID, nil
		}
	}
	return "", err
}

// GetClientServiceAccount ...
func (ic *iamClient) GetClientServiceAccount(accessToken string, internalClient string) (*gocloak.User, error) {
	serviceAccountUser, err := ic.kcClient.GetClientServiceAccount(ic.ctx, accessToken, ic.realmConfig.Realm, internalClient)
	if err != nil {
		return nil, fmt.Errorf("getting client ServiceAccount: %w", err)

	}
	return serviceAccountUser, nil
}

// UpdateServiceAccountUser ...
func (ic *iamClient) UpdateServiceAccountUser(accessToken string, serviceAccountUser gocloak.User) error {
	err := ic.kcClient.UpdateUser(ic.ctx, accessToken, ic.realmConfig.Realm, serviceAccountUser)
	if err != nil {
		return fmt.Errorf("updating service account user: %w", err)
	}
	return nil
}

// GetClients ...
func (ic *iamClient) GetClients(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		First:                &first,
		SearchableAttributes: &attribute,
	}

	if max == 0 {
		max = ic.config.MaxLimitForGetClients
	}

	if max > 0 {
		params.Max = &max
	}

	clients, err := ic.kcClient.GetClients(ic.ctx, accessToken, ic.realmConfig.Realm, params)
	if err != nil {
		return nil, fmt.Errorf("getting clients: %w", err)
	}
	return clients, nil
}

// IsSameOrg ...
func (ic *iamClient) IsSameOrg(client *gocloak.Client, orgID string) bool {
	if orgID == "" {
		return false
	}
	attributes := *client.Attributes
	return attributes[OrgKey] == orgID
}

// IsOwner ...
func (ic *iamClient) IsOwner(client *gocloak.Client, userID string) bool {
	if userID == "" {
		return false
	}
	attributes := *client.Attributes
	if rhUserID, found := attributes[UserKey]; found {
		return rhUserID == userID
	}
	return false
}

// RegenerateClientSecret ...
func (ic *iamClient) RegenerateClientSecret(accessToken string, id string) (*gocloak.CredentialRepresentation, error) {
	credRep, err := ic.kcClient.RegenerateClientSecret(ic.ctx, accessToken, ic.realmConfig.Realm, id)
	if err != nil {
		return nil, fmt.Errorf("regenerating client secrets: %w", err)
	}
	return credRep, nil
}

// GetRealmRole ...
func (ic *iamClient) GetRealmRole(accessToken string, roleName string) (*gocloak.Role, error) {
	r, err := ic.kcClient.GetRealmRole(ic.ctx, accessToken, ic.realmConfig.Realm, roleName)
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting realm role: %w", err)
	}
	return r, nil
}

// CreateRealmRole ...
func (ic *iamClient) CreateRealmRole(accessToken string, roleName string) (*gocloak.Role, error) {
	r := &gocloak.Role{
		Name: &roleName,
	}
	_, err := ic.kcClient.CreateRealmRole(ic.ctx, accessToken, ic.realmConfig.Realm, *r)
	if err != nil {
		return nil, fmt.Errorf("creating realm role: %w", err)
	}
	// for some reason, the internal id of the role is not returned by iamClient.CreateRealmRole, so we have to get the role again to get the full details
	r, err = ic.kcClient.GetRealmRole(ic.ctx, accessToken, ic.realmConfig.Realm, roleName)
	if err != nil {
		return nil, fmt.Errorf("getting realm role: %w", err)
	}
	return r, nil
}

// UserHasRealmRole ...
func (ic *iamClient) UserHasRealmRole(accessToken string, userID string, roleName string) (*gocloak.Role, error) {
	roles, err := ic.kcClient.GetRealmRolesByUserID(ic.ctx, accessToken, ic.realmConfig.Realm, userID)
	if err != nil {
		return nil, fmt.Errorf("getting realm roles by UserID: %w", err)
	}
	for _, r := range roles {
		if *r.Name == roleName {
			return r, nil
		}
	}
	return nil, nil
}

// AddRealmRoleToUser ...
func (ic *iamClient) AddRealmRoleToUser(accessToken string, userID string, role gocloak.Role) error {
	roles := []gocloak.Role{role}
	err := ic.kcClient.AddRealmRoleToUser(ic.ctx, accessToken, ic.realmConfig.Realm, userID, roles)
	if err != nil {
		return fmt.Errorf("adding realm to user: %w", err)
	}
	return nil
}

func isNotFoundError(err error) bool {
	if e, ok := err.(*gocloak.APIError); ok {
		return e.Code == http.StatusNotFound
	}
	return false
}
