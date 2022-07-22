package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// Provider ...
type Provider string

// CompleteServiceAccountRequest ...
type CompleteServiceAccountRequest struct {
	Owner          string
	OwnerAccountID string
	OrgID          string
	ClientID       string
	Name           string
	Description    string
}

// IAMService ...
//go:generate moq -out iam_service_moq.go . IAMService
type IAMService interface {
	GetConfig() *iam.IAMConfig
	GetRealmConfig() *iam.IAMRealmConfig
	RegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) *errors.ServiceError
}

// NewIAMService ...
func NewIAMService(config *iam.IAMConfig) IAMService {
	return &redhatssoService{
		client: redhatsso.NewSSOClient(config, config.RedhatSSORealm),
	}
}
