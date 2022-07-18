package metrics

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

func ConfigProviders() di.Option {
	return di.Options(
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: RegisterVersionMetrics,
		}),
	)
}
