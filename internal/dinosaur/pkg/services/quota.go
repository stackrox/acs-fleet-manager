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
	// HasQuotaAllowance checks if allowed quota is not zero for the given instance type
	HasQuotaAllowance(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError)
	// ReserveQuota reserves a quota for a user and return the reservation id or an error in case of failure
	ReserveQuota(ctx context.Context, dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType, forceBillingModel string, forceProduct string) (string, *errors.ServiceError)
	// DeleteQuota deletes a reserved quota
	DeleteQuota(subscriptionID string) *errors.ServiceError
}
