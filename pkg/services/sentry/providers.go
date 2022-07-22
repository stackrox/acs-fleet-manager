package sentry

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/fleet-manager-pkg/pkg/services/sentry"
)

// ConfigProviders ...
func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(sentry.NewConfig, di.As(new(environments.ConfigModule))),
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: sentry.Initialize,
		}),
	)
}
