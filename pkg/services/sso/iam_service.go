// Package sso ...
package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/serviceaccounts"
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
//
//go:generate moq -out iam_service_moq.go . IAMService
type IAMService interface {
	RegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) (*api.ServiceAccount, *errors.ServiceError)
	DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) *errors.ServiceError
}

// NewIAMService ...
func NewIAMService(config *iam.IAMConfig) IAMService {
	return &redhatssoService{
		serviceAccountsAPI: serviceaccounts.NewServiceAccountsAPI(config.RedhatSSORealm),
	}
}
