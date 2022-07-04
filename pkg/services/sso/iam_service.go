package sso

import (
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

//go:generate moq -out iam_service_moq.go . IAMService
type IAMService interface {
	GetConfig() *iam.IAMConfig
	GetRealmConfig() *iam.IAMRealmConfig
	RegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterId string) *errors.ServiceError
	GetAcsClientSecret(clientId string) (string, *errors.ServiceError)
	CreateServiceAccountInternal(request CompleteServiceAccountRequest) (*api.ServiceAccount, *errors.ServiceError)
	DeleteServiceAccountInternal(clientId string) *errors.ServiceError
}

func NewKeycloakServiceBuilder() KeycloakServiceBuilderSelector {
	return &keycloakServiceBuilderSelector{}
}
