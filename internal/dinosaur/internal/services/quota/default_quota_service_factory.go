package quota

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/quota_management"
)

// DefaultQuotaServiceFactory the default implementation for ProviderFactory
type DefaultQuotaServiceFactory struct {
	quoataServiceContainer map[api.QuotaType]services.QuotaService
}

func NewDefaultQuotaServiceFactory(
	amsClient ocm.AMSClient,
	connectionFactory *db.ConnectionFactory,
	quotaManagementListConfig *quota_management.QuotaManagementListConfig,
) services.QuotaServiceFactory {
	quoataServiceContainer := map[api.QuotaType]services.QuotaService{
		api.AMSQuotaType:                 &amsQuotaService{amsClient: amsClient},
		api.QuotaManagementListQuotaType: &QuotaManagementListService{connectionFactory: connectionFactory, quotaManagementList: quotaManagementListConfig},
	}
	return &DefaultQuotaServiceFactory{quoataServiceContainer: quoataServiceContainer}
}

func (factory *DefaultQuotaServiceFactory) GetQuotaService(quoataType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
	if quoataType == api.UndefinedQuotaType {
		quoataType = api.QuotaManagementListQuotaType
	}

	quotaService, ok := factory.quoataServiceContainer[quoataType]
	if !ok {
		return nil, errors.GeneralError("invalid quota service type: %v", quoataType)
	}

	return quotaService, nil
}
