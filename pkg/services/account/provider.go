package account

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.Func(ServiceProviders)),
	)
}

func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(NewAccount),
	)
}

func NewAccount(ocmConfig *ocm.OCMConfig) AccountService {
	if ocmConfig.EnableMock {
		return NewMockAccountService()
	} else {
		connection, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsUrl)
		if err != nil {
			logger.Logger.Error(err)
		}
		return NewAccountService(connection)
	}
}
