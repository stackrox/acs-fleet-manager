package sso

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

var _ IAMService = &redhatssoService{}

type redhatssoService struct {
	client redhatsso.SSOClient
}

// GetConfig ...
func (r *redhatssoService) GetConfig() *iam.IAMConfig {
	return r.client.GetConfig()
}

// GetRealmConfig ...
func (r *redhatssoService) GetRealmConfig() *iam.IAMRealmConfig {
	return r.client.GetRealmConfig()
}

// RegisterAcsFleetshardOperatorServiceAccount ...
func (r *redhatssoService) RegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) (*api.ServiceAccount, *errors.ServiceError) {
	accessToken, err := r.getToken()
	if err != nil {
		return nil, err
	}

	return r.registerAgentServiceAccount(accessToken, agentClusterID)
}

func (r *redhatssoService) registerAgentServiceAccount(accessToken string, agentClusterID string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Registering agent service account with cluster: %s", agentClusterID)
	svcData, err := r.client.CreateServiceAccount(accessToken, agentClusterID, fmt.Sprintf("service account for agent on cluster %s", agentClusterID))
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to create agent service account")
	}

	glog.V(5).Infof("Agent service account registered with cluster: %s", agentClusterID)
	return convertServiceAccountDataToAPIServiceAccount(&svcData), nil
}

// DeRegisterAcsFleetshardOperatorServiceAccount ...
func (r *redhatssoService) DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) *errors.ServiceError {
	glog.V(5).Infof("Deregistering ACS fleetshard operator service account with cluster: %s", agentClusterID)
	accessToken, tokenErr := r.getToken()
	if tokenErr != nil {
		return tokenErr
	}

	_, found, err := r.client.GetServiceAccount(accessToken, agentClusterID)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterID)
	}
	if !found {
		// if the account to be deleted does not exist, we simply exit with no errors
		glog.V(5).Infof("ACS fleetshard operator service account not found")
		return nil
	}

	err = r.client.DeleteServiceAccount(accessToken, agentClusterID)
	if err != nil {
		return errors.NewWithCause(errors.ErrorFailedToDeleteServiceAccount, err, "Failed to delete service account: %s", agentClusterID)
	}

	glog.V(5).Infof("ACS fleetshard operator service account deregistered with cluster: %s", agentClusterID)
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
