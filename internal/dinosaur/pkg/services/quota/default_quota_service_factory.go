package quota

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"
	"sync"
)

// DefaultQuotaServiceFactory the default implementation for ProviderFactory
type DefaultQuotaServiceFactory struct {
	quotaServiceContainer map[api.QuotaType]QuotaService
}

var (
	onceQuotaServiceFactory sync.Once
	quotaServiceFactory     QuotaServiceFactory
)

// SingletonDefaultQuotaServiceFactory returns a services.QuotaServiceFactory
func SingletonDefaultQuotaServiceFactory() QuotaServiceFactory {
	onceQuotaServiceFactory.Do(func() {
		quotaServiceFactory = NewDefaultQuotaServiceFactory(
			ocm.SingletonAMSClient(),
			db.SingletonConnectionFactory(),
			quotamanagement.GetQuotaManagementListConfig(),
		)
	})
	return quotaServiceFactory
}

// NewDefaultQuotaServiceFactory ...
func NewDefaultQuotaServiceFactory(
	amsClient ocm.AMSClient,
	connectionFactory *db.ConnectionFactory,
	quotaManagementListConfig *quotamanagement.QuotaManagementListConfig,
) QuotaServiceFactory {
	quotaServiceContainer := map[api.QuotaType]QuotaService{
		api.AMSQuotaType:                 &amsQuotaService{amsClient: amsClient},
		api.QuotaManagementListQuotaType: &QuotaManagementListService{connectionFactory: connectionFactory, quotaManagementList: quotaManagementListConfig},
	}
	return &DefaultQuotaServiceFactory{quotaServiceContainer: quotaServiceContainer}
}

// GetQuotaService ...
func (factory *DefaultQuotaServiceFactory) GetQuotaService(quotaType api.QuotaType) (QuotaService, *errors.ServiceError) {
	if quotaType == api.UndefinedQuotaType {
		quotaType = api.QuotaManagementListQuotaType
	}

	quotaService, ok := factory.quotaServiceContainer[quotaType]
	if !ok {
		return nil, errors.GeneralError("invalid quota service type: %v", quotaType)
	}

	return quotaService, nil
}
