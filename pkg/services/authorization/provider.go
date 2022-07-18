package authorization

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
		di.Provide(NewAuthorization),
	)
}

func NewAuthorization(ocmConfig *ocm.OCMConfig) Authorization {
	if ocmConfig.EnableMock {
		return NewMockAuthorization()
	} else {
		connection, _, err := ocm.NewOCMConnection(ocmConfig, ocmConfig.AmsUrl)
		if err != nil {
			logger.Logger.Error(err)
		}
		return NewOCMAuthorization(connection)
	}
}
