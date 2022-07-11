package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
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
}

func NewIAMService(config *iam.IAMConfig) IAMService {
	return &redhatssoService{
		client: redhatsso.NewSSOClient(config, config.RedhatSSORealm),
	}
}
