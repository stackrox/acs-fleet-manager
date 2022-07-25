package account

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/fleet-manager-pkg/pkg/services/account"
)

// ConfigProviders ...
func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.Func(ServiceProviders)),
	)
}

// ServiceProviders ...
func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(NewAccount),
	)
}

// NewAccount ...
func NewAccount(ocmConfig *ocm.OCMConfig) account.AccountService {
	if ocmConfig.EnableMock {
		return account.NewMockAccountService()
	}
	connection, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsURL)
	if err != nil {
		logger.Logger.Error(err)
	}
	return account.NewAccountService(connection)
}
