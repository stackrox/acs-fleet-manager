package sentry

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
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
