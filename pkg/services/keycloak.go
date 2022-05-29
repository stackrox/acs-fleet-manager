package services

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Nerzal/gocloak/v11"
	"github.com/golang/glog"

	"github.com/google/uuid"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

const (
	rhOrgId                  = "rh-org-id"
	rhUserId                 = "rh-user-id"
	username                 = "username"
	created_at               = "created_at"
	clusterId                = "fleetshard-operator-cluster-id"
	UserServiceAccountPrefix = "srvc-acct-"
)

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type KeycloakService interface {
	RegisterDinosaurClientInSSO(dinosaurNamespace string, orgId string) (string, *errors.ServiceError)
	RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError)
	DeRegisterClientInSSO(dinosaurNamespace string) *errors.ServiceError
	GetConfig() *keycloak.KeycloakConfig
	GetRealmConfig() *keycloak.KeycloakRealmConfig
	IsDinosaurClientExist(clientId string) *errors.ServiceError
	RegisterFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError
	GetDinosaurClientSecret(clientId string) (string, *errors.ServiceError)
	CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccountInternal(clientId string) *errors.ServiceError
}

type DinosaurKeycloakService KeycloakService
type OsdKeycloakService KeycloakService

type keycloakService struct {
	kcClient keycloak.KcClient
}

type CompleteServiceAccountRequest struct {
	Owner          string
	OwnerAccountId string
	OrgId          string
	ClientId       string
	Name           string
	Description    string
}

var _ KeycloakService = &keycloakService{}

func NewKeycloakServiceWithClient(client keycloak.KcClient) *keycloakService {
	return &keycloakService{
		kcClient: client,
	}
}

func NewKeycloakService(config *keycloak.KeycloakConfig, realmConfig *keycloak.KeycloakRealmConfig) *keycloakService {
	client := keycloak.NewClient(config, realmConfig)
	return &keycloakService{
		kcClient: client,
	}
}

func (kc *keycloakService) RegisterDinosaurClientInSSO(dinosaurClusterName string, orgId string) (string, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to register Dinosaur Client in SSO")
	}
	internalClientId, err := kc.kcClient.IsClientExist(dinosaurClusterName, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", dinosaurClusterName)

	}
	if internalClientId != "" {
		glog.V(5).Infof("Existing Dinosaur Client %s found", dinosaurClusterName)
		secretValue, secretErr := kc.kcClient.GetClientSecret(internalClientId, accessToken)
		if secretErr != nil {
			return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, secretErr, "failed to get sso client secret")
		}
		return secretValue, nil
	}
	rhOrgIdAttributes := map[string]string{
		"owner-rh-org-id": orgId,
	}
	c := keycloak.ClientRepresentation{
		ClientID:                     dinosaurClusterName,
		Name:                         dinosaurClusterName,
		ServiceAccountsEnabled:       true,
		AuthorizationServicesEnabled: false,
		StandardFlowEnabled:          false,
		Attributes:                   rhOrgIdAttributes,
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
	glog.V(5).Infof("Dinosaur Client %s created successfully", dinosaurClusterName)
	return secretValue, nil
}

func (kc *keycloakService) RegisterOSDClusterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to register OSD cluster Client in SSO")
	}

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

	return secretValue, nil
}

func (kc *keycloakService) DeRegisterClientInSSO(clientId string) *errors.ServiceError {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to deregister OSD cluster client in SSO")
	}
	internalClientID, _ := kc.kcClient.IsClientExist(clientId, accessToken)
	glog.V(5).Infof("Existing Dinosaur Client %s found", clientId)
	if internalClientID == "" {
		return nil
	}
	err := kc.kcClient.DeleteClient(internalClientID, accessToken)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteSSOClient, err, "failed to delete the sso client")
	}
	glog.V(5).Infof("Dinosaur Client %s deleted successfully", clientId)
	return nil
}

func (kc *keycloakService) GetConfig() *keycloak.KeycloakConfig {
	return kc.kcClient.GetConfig()
}

func (kc *keycloakService) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return kc.kcClient.GetRealmConfig()
}

func (kc keycloakService) IsDinosaurClientExist(clientId string) *errors.ServiceError {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to check Dinosaur client existence")
	}
	_, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clusterId)
	}
	return nil
}

func (kc keycloakService) GetDinosaurClientSecret(clientId string) (string, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to get Dinosaur client secret")
	}
	internalClientID, err := kc.kcClient.IsClientExist(clientId, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClient, err, "failed to get sso client with id: %s", clusterId)
	}
	clientSecret, err := kc.kcClient.GetClientSecret(internalClientID, accessToken)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
	}
	return clientSecret, nil
}

func (kc *keycloakService) CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to create service account")
	}
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
	glog.V(5).Infof("service account clientId = %s created for user = %s", serviceAcc.ClientID, request.Owner)
	return serviceAcc, nil
}

func (kc *keycloakService) DeleteServiceAccountInternal(serviceAccountId string) *errors.ServiceError {
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil { //5xx
		return errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to delete service account")
	}

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

	glog.V(5).Infof("deleted service account clientId = %s", id)
	return nil
}

func (kc *keycloakService) RegisterFleetshardOperatorServiceAccount(agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	serviceAccountId := buildAgentOperatorServiceAccountId(agentClusterId)
	return kc.registerAgentServiceAccount(clusterId, serviceAccountId, agentClusterId, roleName)
}

func (kc *keycloakService) DeRegisterFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	accessToken, err := kc.kcClient.GetToken()
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to deregister service account")

	}
	serviceAccountId := buildAgentOperatorServiceAccountId(agentClusterId)
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
	return nil
}

func (kc *keycloakService) registerAgentServiceAccount(clusterId string, serviceAccountId string, agentClusterId string, roleName string) (*api.ServiceAccount, *errors.ServiceError) {
	//get keycloak service client id token
	accessToken, tokenErr := kc.kcClient.GetToken()
	if tokenErr != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, tokenErr, "failed to register agent service account")
	}

	role, err := kc.getRealmRole(accessToken, roleName)
	if err != nil {
		return nil, err
	}

	protocolMapper := kc.kcClient.CreateProtocolMapperConfig(clusterId)

	attributes := map[string]string{
		clusterId: agentClusterId,
	}

	if kc.GetConfig().KeycloakClientExpire {
		attributes["expire_date"] = time.Now().Local().Add(time.Hour * time.Duration(2)).Format(time.RFC3339)
	}

	c := keycloak.ClientRepresentation{
		ClientID:               serviceAccountId,
		Name:                   serviceAccountId,
		Description:            fmt.Sprintf("service account for agent on cluster %s", agentClusterId),
		ServiceAccountsEnabled: true,
		StandardFlowEnabled:    false,
		ProtocolMappers:        protocolMapper,
		Attributes:             attributes,
	}
	account, err := kc.createServiceAccountIfNotExists(accessToken, c)
	if err != nil {
		return nil, err
	}
	serviceAccountUser, getErr := kc.kcClient.GetClientServiceAccount(accessToken, account.ID)
	if getErr != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, getErr, "failed to get agent service account")
	}
	if serviceAccountUser.Attributes == nil || !gocloak.UserAttributeContains(*serviceAccountUser.Attributes, clusterId, agentClusterId) {
		glog.V(10).Infof("Client %s has no attribute %s, set it", serviceAccountId, clusterId)
		serviceAccountUser.Attributes = &map[string][]string{
			clusterId: {agentClusterId},
		}
		updateErr := kc.kcClient.UpdateServiceAccountUser(accessToken, *serviceAccountUser)
		if updateErr != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, updateErr, "failed to update agent service account")
		}
	}
	glog.V(5).Infof("Attribute %s is added for client %s", clusterId, serviceAccountId)
	hasRole, checkErr := kc.kcClient.UserHasRealmRole(accessToken, *serviceAccountUser.ID, roleName)
	if checkErr != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, checkErr, "failed to update agent service account")
	}
	if hasRole == nil {
		glog.V(10).Infof("Client %s has no role %s, adding", serviceAccountId, roleName)
		addRoleErr := kc.kcClient.AddRealmRoleToUser(accessToken, *serviceAccountUser.ID, *role)
		if addRoleErr != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, addRoleErr, "failed to add role to agent service account")
		}
	}
	glog.V(5).Infof("Client %s created successfully", serviceAccountId)
	return account, nil
}

func (kc *keycloakService) getRealmRole(token string, roleName string) (*gocloak.Role, *errors.ServiceError) {
	glog.V(5).Infof("Fetching realm role %s", roleName)
	role, err := kc.kcClient.GetRealmRole(token, roleName)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get realm role")
	}
	if role == nil {
		glog.V(10).Infof("No existing realm role %s found", roleName)
	}
	glog.V(5).Infof("Realm role %s found", roleName)
	return role, nil
}

func (kc *keycloakService) createServiceAccountIfNotExists(token string, clientRep keycloak.ClientRepresentation) (*api.ServiceAccount, *errors.ServiceError) {
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
		glog.V(10).Infof("Existing client found for %s, internalId=%s", clientRep.ClientID, *client.ID)
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

func buildAgentOperatorServiceAccountId(agentClusterId string) string {
	return fmt.Sprintf("fleetshard-agent-%s", agentClusterId)
}

func NewUUID() string {
	return uuid.New().String()
}
