package authorization

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
		di.Provide(NewAuthorization),
	)
}

// NewAuthorization ...
func NewAuthorization(ocmConfig *ocm.OCMConfig) Authorization {
	if ocmConfig.EnableMock {
		return NewMockAuthorization()
	}
	connection, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsURL)
	if err != nil {
		logger.Logger.Error(err)
	}
	return NewOCMAuthorization(connection)
}
