package sso

import (
	"context"
	"fmt"
	"github.com/Nerzal/gocloak/v8"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

const (
	rhOrgId                            = "rh-org-id"
	rhUserId                           = "rh-user-id"
	username                           = "username"
	created_at                         = "created_at"
	acsClusterId                       = "acs-fleetshard-operator-cluster-id"
	connectorClusterId                 = "connector-fleetshard-operator-cluster-id"
	UserServiceAccountPrefix           = "srvc-acct-"
	acsAgentServiceAccountPrefix       = "acs-fleetshard"
	connectorAgentServiceAccountPrefix = "connector-fleetshard"
)

type DinosaurKeycloakService KeycloakService
type OsdKeycloakService OSDKeycloakService

type masService struct {
	kcClient keycloak.KcClient
}

var _ keycloakServiceInternal = &masService{}

func newKeycloakService(config *keycloak.KeycloakConfig, realmConfig *keycloak.KeycloakRealmConfig) KeycloakService {
	client := keycloak.NewClient(config, realmConfig)
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &masService{
			kcClient: client,
		},
	}
}

func (kc *masService) DeRegisterClientInSSO(accessToken string, clientId string) *errors.ServiceError {
	internalClientID, _ := kc.kcClient.IsClientExist(clientId, accessToken)
	glog.V(5).Infof("Existing ACS Client %s found", clientId)
	if internalClientID == "" {
		return nil
	}
	err := kc.kcClient.DeleteClient(internalClientID, accessToken)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteSSOClient, err, "failed to delete the sso client")
	}
	glog.V(5).Infof("ACS Client %s with internal id of %s deleted successfully", clientId, internalClientID)
	return nil
}

func NewKeycloakServiceWithClient(client keycloak.KcClient) KeycloakService {
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &masService{
			kcClient: client,
		},
	}
}

func (kc *masService) RegisterClientInSSO(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	internalClientId, err := kc.kcClient.IsClientExist(clusterId, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clusterId)
	}

	if internalClientId != "" {
		secretValue, _ := kc.kcClient.GetClientSecret(internalClientId, accessToken)
		return secretValue, nil
	}

	c := keycloak.ClientRepresentation{
		ClientID:                     clusterId,
		Name:                         clusterId,
		ServiceAccountsEnabled:       false,
		AuthorizationServicesEnabled: false,
		StandardFlowEnabled:          true,
		RedirectURIs:                 &[]string{clusterOathCallbackURI},
	}

	clientConfig := kc.kcClient.ClientConfig(c)
	internalClient, err := kc.kcClient.CreateClient(clientConfig, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToCreateSSOClient, err, "failed to create sso client")
	}
	secretValue, err := kc.kcClient.GetClientSecret(internalClient, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
	}
	glog.V(5).Infof("ACS Client %s created successfully with internal id = %s", clusterId, internalClient)
	return secretValue, nil
}

func (kc *masService) GetConfig() *keycloak.KeycloakConfig {
	return kc.kcClient.GetConfig()
}

func (kc *masService) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return kc.kcClient.GetRealmConfig()
}

func (kc masService) IsAcsClientExist(accessToken string, clientId string) *errors.ServiceError {
	_, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clientId)
	}
	return nil
}

func (kc masService) GetAcsClientSecret(accessToken string, clientId string) (string, *errors.ServiceError) {
	internalClientID, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clientId)
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClientID, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
	}
	return clientSecret, nil
}

func (kc *masService) CreateServiceAccount(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx) //http requester's info
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}
	orgId, _ := claims.GetOrgId()
	ownerAccountId, _ := claims.GetAccountId()
	owner, _ := claims.GetUsername()
	isAllowed, err := kc.checkAllowedServiceAccountsLimits(accessToken, kc.GetConfig().MaxAllowedServiceAccounts, orgId)
	if err != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to create service account")
	}
	if !isAllowed { //4xx over requesters' limit
		return nil, errors.MaxLimitForServiceAccountReached("Max allowed number:%d of service accounts for user in org:%s has reached", kc.GetConfig().MaxAllowedServiceAccounts, orgId)
	}
	return kc.CreateServiceAccountInternal(accessToken, CompleteServiceAccountRequest{
		Owner:          owner,
		OwnerAccountId: ownerAccountId,
		OrgId:          orgId,
		ClientId:       kc.buildServiceAccountIdentifier(),
		Name:           serviceAccountRequest.Name,
		Description:    serviceAccountRequest.Description,
	})
}

func (kc *masService) CreateServiceAccountInternal(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("creating service accounts: user = %s", request.Owner)
	createdAt := time.Now().Format(time.RFC3339)
	rhAccountID := map[string][]string{
		rhOrgId:  {request.OrgId},
		rhUserId: {request.OwnerAccountId},
		username: {request.Owner},
	}
	rhOrgIdAttributes := map[string]string{
		rhOrgId:    request.OrgId,
		rhUserId:   request.OwnerAccountId,
		username:   request.Owner,
		created_at: createdAt,
	}
	OrgIdProtocolMapper := kc.kcClient.CreateProtocolMapperConfig(rhOrgId)
	userIdProtocolMapper := kc.kcClient.CreateProtocolMapperConfig(rhUserId)
	userProtocolMapper := kc.kcClient.CreateProtocolMapperConfig(username)
	protocolMapper := append(OrgIdProtocolMapper, userIdProtocolMapper...)
	protocolMapper = append(protocolMapper, userProtocolMapper...)

	c := keycloak.ClientRepresentation{
		ClientID:               request.ClientId,
		Name:                   request.Name,
		Description:            request.Description,
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
		ProtocolMappers:        protocolMapper,
		Attributes:             rhOrgIdAttributes,
	}

	serviceAcc, creationErr := kc.createServiceAccountIfNotExists(accessToken, c)
	if creationErr != nil { //5xx
		return nil, creationErr
	}
	serviceAccountUser, getErr := kc.kcClient.GetClientServiceAccount(accessToken, serviceAcc.ID)
	if getErr != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorFailedToGetServiceAccount, getErr, "failed to fetch service account")
	}
	serviceAccountUser.Attributes = &rhAccountID
	serAccUser := *serviceAccountUser
	//step 2
	updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, serAccUser)
	if updateErr != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, updateErr, "failed to create service account")
	}
	serviceAcc.Owner = request.Owner
	creationTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		creationTime = time.Time{}
	}
	serviceAcc.CreatedAt = creationTime
	glog.V(5).Infof("service account clientId = %s and internal id = %s created for user = %s", serviceAcc.ClientID, serviceAcc.ID, request.Owner)
	return serviceAcc, nil
}

func (kc *masService) buildServiceAccountIdentifier() string {
	return UserServiceAccountPrefix + NewUUID()
}

func (kc *masService) ListServiceAcc(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil { //4xx
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}
	orgId, _ := claims.GetOrgId()
	searchAtt := fmt.Sprintf("rh-org-id:%s", orgId)
	clients, err := kc.kcClient.GetClients(accessToken, first, max, searchAtt)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to collect service accounts")
	}

	var sa []api.ServiceAccount
	for _, client := range clients {
		acc := api.ServiceAccount{}
		attributes := client.Attributes
		att := *attributes
		if !strings.HasPrefix(shared.SafeString(client.ClientID), UserServiceAccountPrefix) {
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, att["created_at"])
		if err != nil {
			createdAt = time.Time{}
		}
		acc.ID = *client.ID
		acc.Owner = att["username"]
		acc.CreatedAt = createdAt
		acc.ClientID = *client.ClientID
		acc.Name = shared.SafeString(client.Name)
		acc.Description = shared.SafeString(client.Description)
		sa = append(sa, acc)
	}
	return sa, nil
}

func (kc *masService) DeleteServiceAccount(accessToken string, ctx context.Context, id string) *errors.ServiceError {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil { //4xx
		return errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}
	//get service account info with keycloak service client id token
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil { //5xx or 4xx
		return handleKeyCloakGetClientError(err, id)
	}

	if !strings.HasPrefix(shared.SafeString(c.ClientID), UserServiceAccountPrefix) {
		return errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found %s", id)
	}

	orgId, _ := claims.GetOrgId()
	userId, _ := claims.GetAccountId()
	owner, _ := claims.GetUsername()
	if kc.kcClient.IsSameOrg(c, orgId) && (kc.kcClient.IsOwner(c, userId) || claims.IsOrgAdmin()) {
		err = kc.kcClient.DeleteClient(id, accessToken) //id existence checked
		if err != nil {                                 //5xx
			return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "failed to delete service account")
		}
		glog.V(5).Infof("deleted service account clientId = %s and internal id = %s owned by user = %s", shared.SafeString(c.ClientID), id, owner)
		return nil
	}

	return errors.NewWithCause(errors.ErrorForbidden, nil, "failed to delete service account")
}

func (kc *masService) DeleteServiceAccountInternal(accessToken string, serviceAccountId string) *errors.ServiceError {
	id, err := kc.kcClient.IsClientExist(serviceAccountId, accessToken)
	if err != nil { //5xx ou 404
		keyErr, _ := err.(gocloak.APIError)
		if keyErr.Code == http.StatusNotFound {
			return nil // consider already deleted
		}
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", serviceAccountId)
	}

	err = kc.kcClient.DeleteClient(id, accessToken)
	if err != nil {
		keyErr, ok := err.(gocloak.APIError)
		if ok && keyErr.Code != http.StatusNotFound { // consider already deleted
			return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "failed to delete service account")
		}
	}

	glog.V(5).Infof("deleted service account clientId = %s and internal id = %s", serviceAccountId, id)
	return nil
}

func (kc *masService) ResetServiceAccountCredentials(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil { //4xx
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}
	c, err := kc.kcClient.GetClientById(id, accessToken)
	if err != nil { //5xx or 4xx
		return nil, handleKeyCloakGetClientError(err, id)
	}

	if !strings.HasPrefix(shared.SafeString(c.ClientID), UserServiceAccountPrefix) {
		return nil, errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found %s", id)
	}

	//http request's info
	orgId, _ := claims.GetOrgId()
	userId, _ := claims.GetAccountId()
	if kc.kcClient.IsSameOrg(c, orgId) && (kc.kcClient.IsOwner(c, userId) || claims.IsOrgAdmin()) {
		credRep, err := kc.kcClient.RegenerateClientSecret(accessToken, id)
		if err != nil { //5xx
			return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to reset service account credentials")
		}
		value := *credRep.Value
		attributes := c.Attributes
		att := *attributes
		createdAt, err := time.Parse(time.RFC3339, att["created_at"])
		if err != nil {
			createdAt = time.Time{}
		}
		glog.V(5).Infof("Client %s with internal id = %s updated successfully ", *c.ClientID, *c.ID)
		return &api.ServiceAccount{
			ID:           *c.ID,
			ClientID:     *c.ClientID,
			CreatedAt:    createdAt,
			Owner:        att["username"],
			ClientSecret: value,
			Name:         shared.SafeString(c.Name),
			Description:  shared.SafeString(c.Description),
		}, nil
	} else { //4xx
		return nil, errors.NewWithCause(errors.ErrorForbidden, nil, "failed to reset service account credentials")
	}
}

// return error object for API caller facing funcs: 5xx or 4xx
func handleKeyCloakGetClientError(err error, id string) *errors.ServiceError {
	if keyErr, ok := err.(*gocloak.APIError); ok {
		if keyErr.Code == http.StatusNotFound {
			return errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found %s", id)
		}
	}
	return errors.NewWithCause(errors.ErrorFailedToGetServiceAccount, err, "failed to get the service account %s", id)
}

func (kc *masService) getServiceAccount(accessToken string, ctx context.Context, getClientFunc func(client keycloak.KcClient, accessToken string) (*gocloak.Client, error), key string) (*api.ServiceAccount, *errors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx) //gather http requester info.
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")

	}
	//get service account info with keycloak service client id token
	c, err := getClientFunc(kc.kcClient, accessToken)
	if err != nil { //5xx or 4xx
		return nil, handleKeyCloakGetClientError(err, key)
	}

	if c == nil || !strings.HasPrefix(shared.SafeString(c.ClientID), UserServiceAccountPrefix) {
		return nil, errors.NewWithCause(errors.ErrorServiceAccountNotFound, err, "service account not found %s", key)
	}

	//http requester's info.
	orgId, _ := claims.GetOrgId()
	userId, _ := claims.GetAccountId()
	owner, _ := claims.GetUsername()
	attributes := c.Attributes
	att := *attributes
	createdAt, err := time.Parse(time.RFC3339, att["created_at"])
	if err != nil {
		createdAt = time.Time{}
	}
	if kc.kcClient.IsSameOrg(c, orgId) && kc.kcClient.IsOwner(c, userId) {
		return &api.ServiceAccount{
			ID:          *c.ID,
			ClientID:    *c.ClientID,
			CreatedAt:   createdAt,
			Owner:       owner,
			Name:        shared.SafeString(c.Name),
			Description: shared.SafeString(c.Description),
		}, nil
	} else {
		//http requester doesn't have the permission: 4xx
		return nil, errors.NewWithCause(errors.ErrorForbidden, nil, "failed to get service account")
	}
}

func (kc *masService) GetServiceAccountByClientId(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	return kc.getServiceAccount(accessToken, ctx, func(client keycloak.KcClient, accessToken string) (*gocloak.Client, error) {
		return client.GetClient(clientId, accessToken)
	}, clientId)
}

func (kc *masService) GetServiceAccountById(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	return kc.getServiceAccount(accessToken, ctx, func(client keycloak.KcClient, accessToken string) (*gocloak.Client, error) {
		return client.GetClientById(id, accessToken)
	}, id)
}

func (kc *masService) RegisterAcsFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := buildAgentOperatorServiceAccountId(acsAgentServiceAccountPrefix, agentClusterId)
	return kc.registerAgentServiceAccount(accessToken, serviceAccountId, agentClusterId)
}

func (kc *masService) DeRegisterAcsFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	return kc.deregisterAgentServiceAccount(accessToken, acsAgentServiceAccountPrefix, agentClusterId)
}

func (kc *masService) RegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) { // (agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := buildAgentOperatorServiceAccountId(connectorAgentServiceAccountPrefix, agentClusterId)
	return kc.registerAgentServiceAccount(accessToken, serviceAccountId, agentClusterId)
}

func (kc *masService) DeRegisterConnectorFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError {
	return kc.deregisterAgentServiceAccount(accessToken, connectorAgentServiceAccountPrefix, agentClusterId)
}

func (kc *masService) registerAgentServiceAccount(accessToken string, serviceAccountId string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	c := keycloak.ClientRepresentation{
		ClientID:               serviceAccountId,
		Name:                   serviceAccountId,
		Description:            fmt.Sprintf("service account for agent on cluster %s", agentClusterId),
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
	}
	account, err := kc.createServiceAccountIfNotExists(accessToken, c)
	if err != nil {
		return nil, err
	}
	glog.V(5).Infof("Client %s created successfully with internal id = %s", serviceAccountId, account.ID)
	return account, nil
}

func (kc *masService) deregisterAgentServiceAccount(accessToken string, prefix string, agentClusterId string) *errors.ServiceError {
	serviceAccountId := buildAgentOperatorServiceAccountId(prefix, agentClusterId)
	internalServiceAccountId, err := kc.kcClient.IsClientExist(serviceAccountId, accessToken)
	if err != nil { //5xx
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", serviceAccountId)
	}
	if internalServiceAccountId == "" {
		return nil
	}
	err = kc.kcClient.DeleteClient(internalServiceAccountId, accessToken)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", internalServiceAccountId)
	}
	glog.V(5).Infof("deleted service account clientId = %s and internal id = %s", serviceAccountId, internalServiceAccountId)
	return nil
}

func (kc *masService) createServiceAccountIfNotExists(token string, clientRep keycloak.ClientRepresentation) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Creating service account: clientId = %s", clientRep.ClientID)
	client, err := kc.kcClient.GetClient(clientRep.ClientID, token)
	if err != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to check if client exists.")
	}

	//client exists
	var internalClientId, clientSecret string
	if client == nil {
		glog.V(10).Infof("No exiting client found for %s, creating a new one", clientRep.ClientID)
		clientConfig := kc.kcClient.ClientConfig(clientRep)
		internalClientId, err = kc.kcClient.CreateClient(clientConfig, token)
		if err != nil { //5xx
			return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, err, "failed to create service account")
		}
	} else {
		glog.V(5).Infof("Existing client found for %s with internal id = %s", clientRep.ClientID, *client.ID)
		internalClientId = *client.ID
	}

	clientSecret, err = kc.kcClient.GetClientSecret(internalClientId, token)
	if err != nil { //5xx
		return nil, errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get service account secret")
	}

	serviceAcc := &api.ServiceAccount{
		ID:           internalClientId,
		ClientID:     clientRep.ClientID,
		ClientSecret: clientSecret,
		Name:         clientRep.Name,
		Description:  clientRep.Description,
	}
	return serviceAcc, nil

}

func (kc *masService) checkAllowedServiceAccountsLimits(accessToken string, maxAllowed int, orgId string) (bool, error) {
	glog.V(5).Infof("Check if user is allowed to create service accounts: orgId = %s", orgId)

	if arrays.Contains(kc.GetConfig().ServiceAccounttLimitCheckSkipOrgIdList, orgId) {
		glog.V(5).Infof("orgId = %s , present in service account limits check skip list. No limits on the number of service accounts", orgId)
		return true, nil
	}
	searchAtt := fmt.Sprintf("rh-org-id:%s", orgId)
	clients, err := kc.kcClient.GetClients(accessToken, 0, -1, searchAtt) // return all service accounts attached to the org
	if err != nil {
		return false, err
	}

	serviceAccountCount := 0
	for _, client := range clients {
		if !strings.HasPrefix(shared.SafeString(client.ClientID), UserServiceAccountPrefix) { // filter out internal ones and care about user facing ones for comparison
			continue
		}
		serviceAccountCount++
	}

	glog.V(10).Infof("Existing number of clients found: %d & max allowed: %d, for the orgId: %s", serviceAccountCount, maxAllowed, orgId)
	if serviceAccountCount >= maxAllowed {
		return false, nil //http requester's error
	} else {
		return true, nil
	}
}

func buildAgentOperatorServiceAccountId(prefix string, agentClusterId string) string {
	return fmt.Sprintf("%s-agent-%s", prefix, agentClusterId)
}

func NewUUID() string {
	return uuid.New().String()
}
