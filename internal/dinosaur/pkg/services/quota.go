package services

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// QuotaService ...
//
//go:generate moq -out quotaservice_moq.go . QuotaService
type QuotaService interface {
	// CheckIfQuotaIsDefinedForInstanceType checks if quota is defined for the given instance type
	CheckIfQuotaIsDefinedForInstanceType(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError)
	// ReserveQuota reserves a quota for a user and return the reservation id or an error in case of failure
	ReserveQuota(ctx context.Context, dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError)
	// DeleteQuota deletes a reserved quota
	DeleteQuota(subscriptionID string) *errors.ServiceError
	// IsQuotaEntitlementActive checks if the user/organisation have an active entitlement to the quota used by the
	// given Central instance.
	// It returns true if the user has an active quota entitlement and false if not.
	// It returns false and an error if it encounters any issues while trying to check the quota entitlement status
	IsQuotaEntitlementActive(dinosaur *dbapi.CentralRequest) (bool, error)
}
