// Package sso ...
package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/serviceaccounts"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"sync"
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

var (
	onceIAMService sync.Once
	iamService     IAMService
)

func SingletonIAMService() IAMService {
	onceIAMService.Do(func() {
		iamService = newIAMService(iam.GetIAMConfig())
	})
	return iamService
}

// NewIAMService ...
func newIAMService(config *iam.IAMConfig) IAMService {
	return &redhatssoService{
		serviceAccountsAPI: serviceaccounts.NewServiceAccountsAPI(config.RedhatSSORealm),
	}
}
