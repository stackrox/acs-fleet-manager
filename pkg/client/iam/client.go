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
	OrgKey               = "rh-org-id"
	UserKey              = "rh-user-id"
)

var (
	protocol = "openid-connect"
	mapper   = "oidc-usermodel-attribute-mapper"
)

//go:generate moq -out client_moq.go . KcClient
type IAMClient interface {
	CreateClient(client gocloak.Client, accessToken string) (string, error)
	GetToken() (string, error)
	GetCachedToken(tokenKey string) (string, error)
	DeleteClient(internalClientID string, accessToken string) error
	GetClientSecret(internalClientId string, accessToken string) (string, error)
	GetClient(clientId string, accessToken string) (*gocloak.Client, error)
	IsClientExist(clientId string, accessToken string) (string, error)
	GetConfig() *IAMConfig
	GetRealmConfig() *IAMRealmConfig
	GetClientById(id string, accessToken string) (*gocloak.Client, error)
	ClientConfig(client ClientRepresentation) gocloak.Client
	CreateProtocolMapperConfig(string) []gocloak.ProtocolMapperRepresentation
	GetClientServiceAccount(accessToken string, internalClient string) (*gocloak.User, error)
	UpdateServiceAccountUser(accessToken string, serviceAccountUser gocloak.User) error
	// GetClients returns iam clients using the given method parameters. If max is less than 0, then returns all the clients.
	// If it is 0, then default to using the default max allowed service accounts configuration.
	GetClients(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error)
	IsSameOrg(client *gocloak.Client, orgId string) bool
	IsOwner(client *gocloak.Client, userId string) bool
	RegenerateClientSecret(accessToken string, id string) (*gocloak.CredentialRepresentation, error)
	GetRealmRole(accessToken string, roleName string) (*gocloak.Role, error)
	CreateRealmRole(accessToken string, roleName string) (*gocloak.Role, error)
	UserHasRealmRole(accessToken string, userId string, roleName string) (*gocloak.Role, error)
	AddRealmRoleToUser(accessToken string, userId string, role gocloak.Role) error
}

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

func (kc *iamClient) ClientConfig(client ClientRepresentation) gocloak.Client {
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

func (kc *iamClient) CreateProtocolMapperConfig(name string) []gocloak.ProtocolMapperRepresentation {
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

func (kc *iamClient) CreateClient(client gocloak.Client, accessToken string) (string, error) {
	internalClientID, err := kc.kcClient.CreateClient(kc.ctx, accessToken, kc.realmConfig.Realm, client)
	if err != nil {
		return "", err
	}
	return internalClientID, err
}

func (kc *iamClient) GetClient(clientId string, accessToken string) (*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		ClientID: &clientId,
	}
	clients, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.realmConfig.Realm, params)
	if err != nil {
		return nil, err
	}
	for _, client := range clients {
		if *client.ClientID == clientId {
			return client, nil
		}
	}
	return nil, nil
}

func (kc *iamClient) GetToken() (string, error) {
	options := gocloak.TokenOptions{
		ClientID:     &kc.realmConfig.ClientID,
		GrantType:    &kc.realmConfig.GrantType,
		ClientSecret: &kc.realmConfig.ClientSecret,
	}
	cachedTokenKey := fmt.Sprintf("%s%s", kc.realmConfig.ValidIssuerURI, kc.realmConfig.ClientID)
	cachedToken, _ := kc.GetCachedToken(cachedTokenKey)

	if cachedToken != "" && !shared.IsJWTTokenExpired(cachedToken) {
		return cachedToken, nil
	}
	tokenResp, err := kc.kcClient.GetToken(kc.ctx, kc.realmConfig.Realm, options)
	if err != nil {
		return "", errors.Wrap(err, "failed to get new token from gocloak with error")
	}

	kc.cache.Set(cachedTokenKey, tokenResp.AccessToken, cacheCleanupInterval)
	return tokenResp.AccessToken, nil
}

func (kc *iamClient) GetCachedToken(tokenKey string) (string, error) {
	cachedToken, isCached := kc.cache.Get(tokenKey)
	ct, _ := cachedToken.(string)
	if isCached {
		return ct, nil
	}
	return "", errors.Errorf("failed to retrieve cached token")
}

func (kc *iamClient) GetClientSecret(internalClientId string, accessToken string) (string, error) {
	resp, err := kc.kcClient.GetClientSecret(kc.ctx, accessToken, kc.realmConfig.Realm, internalClientId)
	if err != nil {
		return "", err
	}
	if resp.Value == nil {
		return "", errors.Errorf("failed to retrieve credentials")
	}
	return *resp.Value, err
}

func (kc *iamClient) DeleteClient(internalClientID string, accessToken string) error {
	return kc.kcClient.DeleteClient(kc.ctx, accessToken, kc.realmConfig.Realm, internalClientID)
}

func (kc *iamClient) getClient(clientId string, accessToken string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		ClientID: &clientId,
	}
	client, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.realmConfig.Realm, params)
	if err != nil {
		return nil, err
	}
	return client, err
}

func (kc *iamClient) GetClientById(internalId string, accessToken string) (*gocloak.Client, error) {
	client, err := kc.kcClient.GetClient(kc.ctx, accessToken, kc.realmConfig.Realm, internalId)
	if err != nil {
		return nil, err
	}
	return client, err
}

func (kc *iamClient) GetConfig() *IAMConfig {
	return kc.config
}

func (kc *iamClient) GetRealmConfig() *IAMRealmConfig {
	return kc.realmConfig
}

func (kc *iamClient) IsClientExist(clientId string, accessToken string) (string, error) {
	if clientId == "" {
		return "", errors.New("clientId cannot be empty")
	}
	clients, err := kc.getClient(clientId, accessToken)
	if err != nil {
		return "", err
	}
	for _, client := range clients {
		if *client.ClientID == clientId {
			return *client.ID, nil
		}
	}
	return "", err
}

func (kc *iamClient) GetClientServiceAccount(accessToken string, internalClient string) (*gocloak.User, error) {
	serviceAccountUser, err := kc.kcClient.GetClientServiceAccount(kc.ctx, accessToken, kc.realmConfig.Realm, internalClient)
	if err != nil {
		return nil, err

	}
	return serviceAccountUser, err
}

func (kc *iamClient) UpdateServiceAccountUser(accessToken string, serviceAccountUser gocloak.User) error {
	err := kc.kcClient.UpdateUser(kc.ctx, accessToken, kc.realmConfig.Realm, serviceAccountUser)
	if err != nil {
		return err
	}
	return err
}

func (kc *iamClient) GetClients(accessToken string, first int, max int, attribute string) ([]*gocloak.Client, error) {
	params := gocloak.GetClientsParams{
		First:                &first,
		SearchableAttributes: &attribute,
	}

	if max == 0 {
		max = kc.config.MaxLimitForGetClients
	}

	if max > 0 {
		params.Max = &max
	}

	clients, err := kc.kcClient.GetClients(kc.ctx, accessToken, kc.realmConfig.Realm, params)
	if err != nil {
		return nil, err
	}
	return clients, err
}

func (kc *iamClient) IsSameOrg(client *gocloak.Client, orgId string) bool {
	if orgId == "" {
		return false
	}
	attributes := *client.Attributes
	return attributes[OrgKey] == orgId
}

func (kc *iamClient) IsOwner(client *gocloak.Client, userId string) bool {
	if userId == "" {
		return false
	}
	attributes := *client.Attributes
	if rhUserId, found := attributes[UserKey]; found {
		return rhUserId == userId
	}
	return false
}

func (kc *iamClient) RegenerateClientSecret(accessToken string, id string) (*gocloak.CredentialRepresentation, error) {
	credRep, err := kc.kcClient.RegenerateClientSecret(kc.ctx, accessToken, kc.realmConfig.Realm, id)
	if err != nil {
		return nil, err
	}
	return credRep, err
}

func (kc *iamClient) GetRealmRole(accessToken string, roleName string) (*gocloak.Role, error) {
	r, err := kc.kcClient.GetRealmRole(kc.ctx, accessToken, kc.realmConfig.Realm, roleName)
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return r, err
}

func (kc *iamClient) CreateRealmRole(accessToken string, roleName string) (*gocloak.Role, error) {
	r := &gocloak.Role{
		Name: &roleName,
	}
	_, err := kc.kcClient.CreateRealmRole(kc.ctx, accessToken, kc.realmConfig.Realm, *r)
	if err != nil {
		return nil, err
	}
	// for some reason, the internal id of the role is not returned by iamClient.CreateRealmRole, so we have to get the role again to get the full details
	r, err = kc.kcClient.GetRealmRole(kc.ctx, accessToken, kc.realmConfig.Realm, roleName)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (kc *iamClient) UserHasRealmRole(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
	roles, err := kc.kcClient.GetRealmRolesByUserID(kc.ctx, accessToken, kc.realmConfig.Realm, userId)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		if *r.Name == roleName {
			return r, nil
		}
	}
	return nil, nil
}

func (kc *iamClient) AddRealmRoleToUser(accessToken string, userId string, role gocloak.Role) error {
	roles := []gocloak.Role{role}
	err := kc.kcClient.AddRealmRoleToUser(kc.ctx, accessToken, kc.realmConfig.Realm, userId, roles)
	if err != nil {
		return err
	}
	return nil
}

func isNotFoundError(err error) bool {
	if e, ok := err.(*gocloak.APIError); ok {
		return e.Code == http.StatusNotFound
	}
	return false
}
