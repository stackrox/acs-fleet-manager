package account

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
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
func NewAccount(ocmConfig *ocm.OCMConfig) AccountService {
	if ocmConfig.EnableMock {
		return NewMockAccountService()
	}
	connection, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsURL)
	if err != nil {
		logger.Logger.Error(err)
	}
	return NewAccountService(connection)
}
