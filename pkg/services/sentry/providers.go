package sentry

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// ConfigProviders ...
func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(NewConfig, di.As(new(environments.ConfigModule))),
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: Initialize,
		}),
	)
}
