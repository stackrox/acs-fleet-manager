package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

//go:generate moq -out quotaservice_moq.go . QuotaService
type QuotaService interface {
	// CheckIfQuotaIsDefinedForInstanceType checks if quota is defined for the given instance type
	CheckIfQuotaIsDefinedForInstanceType(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError)
	// ReserveQuota reserves a quota for a user and return the reservation id or an error in case of failure
	ReserveQuota(dinosaur *dbapi.DinosaurRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError)
	// DeleteQuota deletes a reserved quota
	DeleteQuota(subscriptionId string) *errors.ServiceError
}
