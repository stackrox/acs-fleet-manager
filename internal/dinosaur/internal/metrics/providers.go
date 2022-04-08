package metrics

import (
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: RegisterVersionMetrics,
		}),
	)
}
