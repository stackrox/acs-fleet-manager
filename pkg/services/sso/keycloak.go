package sso

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type Provider string

type CompleteServiceAccountRequest struct {
	Owner          string
	OwnerAccountId string
	OrgId          string
	ClientId       string
	Name           string
	Description    string
}

//go:generate moq -out keycloakservice_moq.go . KeycloakService
type IAMService interface {
	GetConfig() *iam.IAMConfig
	GetRealmConfig() *iam.IAMRealmConfig
	CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError)
	RegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError
	GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError)
	GetServiceAccountByClientId(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	GetAcsClientSecret(clientId string) (string, *errors.ServiceError)
	CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccountInternal(clientId string) *errors.ServiceError
}

type OSDKeycloakService interface {
	IAMService
	DeRegisterClientInSSO(namespace string) *errors.ServiceError
	RegisterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError)
}

type keycloakServiceInternal interface {
	DeRegisterClientInSSO(accessToken string, namespace string) *errors.ServiceError
	RegisterClientInSSO(accessToken string, clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError)
	GetConfig() *iam.IAMConfig
	GetRealmConfig() *iam.IAMRealmConfig
	CreateServiceAccount(accessToken string, serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccount(accessToken string, ctx context.Context, clientId string) *errors.ServiceError
	ResetServiceAccountCredentials(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	ListServiceAcc(accessToken string, ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError)
	RegisterAcsFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterAcsFleetshardOperatorServiceAccount(accessToken string, agentClusterId string) *errors.ServiceError
	GetServiceAccountById(accessToken string, ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError)
	GetServiceAccountByClientId(accessToken string, ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError)
	GetAcsClientSecret(accessToken string, clientId string) (string, *errors.ServiceError)
	CreateServiceAccountInternal(accessToken string, request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccountInternal(accessToken string, clientId string) *errors.ServiceError
}

func NewKeycloakServiceBuilder() KeycloakServiceBuilderSelector {
	return &keycloakServiceBuilderSelector{}
}
