package rhacs

import 	(
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/routes"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/environments"
	environments2 "github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/providers"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(

		EnvConfigProviders(),
		providers.CoreConfigProviders(),

		// rhacs service config
		di.Provide(config.NewSupportedProvidersConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),

		// config for CLI commands
		di.Provide(environments2.Func(ServiceProviders)),
	)
}


func EnvConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.NewDevelopmentEnvLoader, di.Tags{"env": environments2.DevelopmentEnv}),
	)
}


func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(routes.NewRouteLoader),
		di.Provide(services.NewCentralService, di.As(new(services.CentralService))),
	)
}

// TODO complete following internal/dinosaur/providers.go