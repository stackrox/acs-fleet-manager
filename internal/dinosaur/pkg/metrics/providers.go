package metrics

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// ConfigProviders ...
func ConfigProviders() di.Option {
	return di.Options(
		di.ProvideValue(environments.AfterCreateServicesHook{
			Func: RegisterVersionMetrics,
		}),
	)
}
