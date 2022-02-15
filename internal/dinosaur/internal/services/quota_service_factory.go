package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
)

// QuotaServiceFactory used to return an instance of QuotaService implementation
//go:generate moq -out quota_service_factory_moq.go . QuotaServiceFactory
type QuotaServiceFactory interface {
	GetQuotaService(quoataType api.QuotaType) (QuotaService, *errors.ServiceError)
}
