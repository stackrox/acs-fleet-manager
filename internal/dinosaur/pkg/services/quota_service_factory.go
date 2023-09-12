package services

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// QuotaServiceFactory used to return an instance of QuotaService implementation
//
//go:generate moq -out quota_service_factory_moq.go . QuotaServiceFactory
type QuotaServiceFactory interface {
	GetQuotaService(quotaType api.QuotaType) (QuotaService, *errors.ServiceError)
}
