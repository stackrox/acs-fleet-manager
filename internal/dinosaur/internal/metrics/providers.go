package metrics

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: RegisterVersionMetrics,
		}),
	)
}
