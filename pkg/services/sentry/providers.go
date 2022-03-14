package sentry

import (
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.Provide(NewConfig, di.As(new(environments.ConfigModule))),
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: Initialize,
		}),
	)
}
