package sso

import (
	"context"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type tokenProvider interface {
	GetToken() (string, error)
}

type keycloakServiceProxy struct {
	accessTokenProvider tokenProvider
	service             keycloakServiceInternal
}

var _ KeycloakService = &keycloakServiceProxy{}
var _ OSDKeycloakService = &keycloakServiceProxy{}

func (r *keycloakServiceProxy) DeRegisterClientInSSO(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeRegisterClientInSSO(token, clientId)
	}
}

func (r *keycloakServiceProxy) RegisterClientInSSO(clusterId string, clusterOathCallbackURI string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		return r.service.RegisterClientInSSO(token, clusterId, clusterOathCallbackURI)
	}
}

func (r *keycloakServiceProxy) GetConfig() *keycloak.KeycloakConfig {
	return r.service.GetConfig()
}

func (r *keycloakServiceProxy) GetRealmConfig() *keycloak.KeycloakRealmConfig {
	return r.service.GetRealmConfig()
}

func (r *keycloakServiceProxy) CreateServiceAccount(serviceAccountRequest *api.ServiceAccountRequest, ctx context.Context) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx); err != nil {
		return nil, err
	} else {
		return r.service.CreateServiceAccount(token, serviceAccountRequest, ctx)
	}
}

func (r *keycloakServiceProxy) DeleteServiceAccount(ctx context.Context, clientId string) *errors.ServiceError {
	if token, err := tokenForServiceAPIHandler(ctx); err != nil {
		return err
	} else {
		return r.service.DeleteServiceAccount(token, ctx, clientId)
	}
}

func (r *keycloakServiceProxy) ResetServiceAccountCredentials(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx); err != nil {
		return nil, err
	} else {
		return r.service.ResetServiceAccountCredentials(token, ctx, clientId)
	}
}

func (r *keycloakServiceProxy) ListServiceAcc(ctx context.Context, first int, max int) ([]api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx); err != nil {
		return nil, err
	} else {
		return r.service.ListServiceAcc(token, ctx, first, max)
	}
}

func (r *keycloakServiceProxy) RegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.RegisterAcsFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}

func (r *keycloakServiceProxy) DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeRegisterAcsFleetshardOperatorServiceAccount(token, agentClusterId)
	}
}

func (r *keycloakServiceProxy) GetServiceAccountById(ctx context.Context, id string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx); err != nil {
		return nil, err
	} else {
		return r.service.GetServiceAccountById(token, ctx, id)
	}
}

func (r *keycloakServiceProxy) GetServiceAccountByClientId(ctx context.Context, clientId string) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := tokenForServiceAPIHandler(ctx); err != nil {
		return nil, err
	} else {
		return r.service.GetServiceAccountByClientId(token, ctx, clientId)
	}
}

func (r *keycloakServiceProxy) GetAcsClientSecret(clientId string) (string, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return "", err
	} else {
		return r.service.GetAcsClientSecret(token, clientId)
	}
}
func (r *keycloakServiceProxy) CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	if token, err := r.retrieveToken(); err != nil {
		return nil, err
	} else {
		return r.service.CreateServiceAccountInternal(token, request)
	}
}
func (r *keycloakServiceProxy) DeleteServiceAccountInternal(clientId string) *errors.ServiceError {
	if token, err := r.retrieveToken(); err != nil {
		return err
	} else {
		return r.service.DeleteServiceAccountInternal(token, clientId)
	}
}

// Utility functions

func (r *keycloakServiceProxy) retrieveToken() (string, *errors.ServiceError) {
	accessToken, tokenErr := r.accessTokenProvider.GetToken()
	if tokenErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, tokenErr, "error getting access token")
	}
	return accessToken, nil
}

func retrieveUserToken(ctx context.Context) (string, *errors.ServiceError) {
	userToken, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "error getting access token")
	}
	token := userToken.Raw
	return token, nil
}

func tokenForServiceAPIHandler(ctx context.Context) (string, *errors.ServiceError) {
	token, err := retrieveUserToken(ctx)
	if err != nil {
		return "", err
	}
	return token, nil
}
