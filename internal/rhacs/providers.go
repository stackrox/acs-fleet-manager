package rhacs

import 	(
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/routes"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/environments"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/handlers"
	environments2 "github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/migrations"
	"github.com/stackrox/acs-fleet-manager/pkg/providers"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Options(

		EnvConfigProviders(),
		providers.CoreConfigProviders(),

		// Command line options
		di.Provide(config.NewDinosaurConfig, di.As(new(environments2.ConfigModule))),

		// rhacs service config
		di.Provide(config.NewSupportedProvidersConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),

		// config for CLI commands
		di.Provide(environments2.Func(ServiceProviders)),
		di.Provide(migrations.New),
	)
}


func EnvConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.NewDevelopmentEnvLoader, di.Tags{"env": environments2.DevelopmentEnv}),
		di.Provide(environments.NewIntegrationEnvLoader, di.Tags{"env": environments2.IntegrationEnv}),
	)
}


func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(routes.NewRouteLoader),
		di.Provide(handlers.NewAuthenticationBuilder),
		di.Provide(services.NewCentralService, di.As(new(services.CentralService))),
		di.Provide(services.NewCloudProvidersService),
	)
}

// TODO complete following internal/dinosaur/providers.go