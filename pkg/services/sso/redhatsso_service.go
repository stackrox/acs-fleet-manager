package sso

import (
	"fmt"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"time"
)

var _ IAMService = &redhatssoService{}

type redhatssoService struct {
	client redhatsso.SSOClient
}

func (r *redhatssoService) GetConfig() *iam.IAMConfig {
	return r.client.GetConfig()
}

func (r *redhatssoService) GetRealmConfig() *iam.IAMRealmConfig {
	return r.client.GetRealmConfig()
}

func (r *redhatssoService) RegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, err := r.getToken()
	if err != nil {
		return nil, err
	}

	return r.registerAgentServiceAccount(accessToken, agentClusterId)
}

func (r *redhatssoService) registerAgentServiceAccount(accessToken string, agentClusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	svcData, err := r.client.CreateServiceAccount(accessToken, agentClusterId, fmt.Sprintf("service account for agent on cluster %s", agentClusterId))
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to create agent service account")
	}
	return convertServiceAccountDataToAPIServiceAccount(&svcData), nil
}

func (r *redhatssoService) DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError {
	accessToken, tokenErr := r.getToken()
	if tokenErr != nil {
		return tokenErr
	}

	if _, found, err := r.client.GetServiceAccount(accessToken, agentClusterId); err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	} else {
		if !found {
			// if the account to be deleted does not exists, we simply exit with no errors
			return nil
		}
	}

	err := r.client.DeleteServiceAccount(accessToken, agentClusterId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterId)
	}
	return nil
}

func (r *redhatssoService) GetAcsClientSecret(clientId string) (string, *errors.ServiceError) {
	accessToken, tokenErr := r.getToken()
	if tokenErr != nil {
		return "", tokenErr
	}

	serviceAccount, found, err := r.client.GetServiceAccount(accessToken, clientId)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
	}
	if !found {
		//if client is found re-generate the client secret.
		svcData, seErr := r.client.RegenerateClientSecret(accessToken, shared.SafeString(serviceAccount.Id))
		if seErr != nil {
			return "", errors.NewWithCause(errors.ErrorFailedToGetSSOClientSecret, err, "failed to get sso client secret")
		}
		return shared.SafeString(svcData.Secret), nil
	}

	return *serviceAccount.Secret, nil
}

func (r *redhatssoService) CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, tokenErr := r.getToken()
	if tokenErr != nil {
		return nil, tokenErr
	}

	svcData, err := r.client.CreateServiceAccount(accessToken, request.ClientId, request.Description)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorFailedToCreateServiceAccount, err, "failed to create service account")
	}
	return convertServiceAccountDataToAPIServiceAccount(&svcData), nil
}
func (r *redhatssoService) DeleteServiceAccountInternal(clientId string) *errors.ServiceError {
	accessToken, tokenErr := r.getToken()
	if tokenErr != nil {
		return tokenErr
	}

	err := r.client.DeleteServiceAccount(accessToken, clientId)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", clientId)
	}
	return nil
}

func (r *redhatssoService) getToken() (string, *errors.ServiceError) {
	accessToken, err := r.client.GetToken()
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "error getting access token")
	}
	return accessToken, nil
}

//// utility functions
func convertServiceAccountDataToAPIServiceAccount(data *serviceaccountsclient.ServiceAccountData) *api.ServiceAccount {
	return &api.ServiceAccount{
		ID:           shared.SafeString(data.Id),
		ClientID:     shared.SafeString(data.ClientId),
		ClientSecret: shared.SafeString(data.Secret),
		Name:         shared.SafeString(data.Name),
		CreatedBy:    shared.SafeString(data.CreatedBy),
		Description:  shared.SafeString(data.Description),
		CreatedAt:    time.Unix(0, shared.SafeInt64(data.CreatedAt)*int64(time.Millisecond)),
	}
}
