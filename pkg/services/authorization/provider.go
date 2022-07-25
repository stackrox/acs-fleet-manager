package authorization

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/fleet-manager-pkg/pkg/services/authorization"
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
func NewAuthorization(ocmConfig *ocm.OCMConfig) authorization.Authorization {
	if ocmConfig.EnableMock {
		return authorization.NewMockAuthorization()
	}
	connection, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsURL)
	if err != nil {
		logger.Logger.Error(err)
	}
	return authorization.NewOCMAuthorization(connection)
}
