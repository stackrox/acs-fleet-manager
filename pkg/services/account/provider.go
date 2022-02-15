package account

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/logger"
	"github.com/goava/di"
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
