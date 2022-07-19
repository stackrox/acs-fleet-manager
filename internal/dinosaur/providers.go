package dinosaur

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/cluster"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/dinosaur"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/cmd/observatorium"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/migrations"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/routes"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services/quota"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers/dinosaur_mgrs"
	observatoriumClient "github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	environments2 "github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/providers"
)

// EnvConfigProviders ...
func EnvConfigProviders() di.Option {
	return di.Options(
		di.Provide(environments.NewDevelopmentEnvLoader, di.Tags{"env": environments2.DevelopmentEnv}),
		di.Provide(environments.NewProductionEnvLoader, di.Tags{"env": environments2.ProductionEnv}),
		di.Provide(environments.NewStageEnvLoader, di.Tags{"env": environments2.StageEnv}),
		di.Provide(environments.NewIntegrationEnvLoader, di.Tags{"env": environments2.IntegrationEnv}),
		di.Provide(environments.NewTestingEnvLoader, di.Tags{"env": environments2.TestingEnv}),
	)
}

// ConfigProviders ...
func ConfigProviders() di.Option {
	return di.Options(

		EnvConfigProviders(),
		providers.CoreConfigProviders(),

		// Configuration for the Dinosaur service...
		di.Provide(config.NewAWSConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewSupportedProvidersConfig, di.As(new(environments2.ConfigModule)), di.As(new(environments2.ServiceValidator))),
		di.Provide(observatoriumClient.NewObservabilityConfigurationConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewDinosaurConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewDataplaneClusterConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewFleetshardConfig, di.As(new(environments2.ConfigModule))),

		// Additional CLI subcommands
		di.Provide(cluster.NewClusterCommand),
		di.Provide(dinosaur.NewDinosaurCommand),
		di.Provide(cloudprovider.NewCloudProviderCommand),
		di.Provide(observatorium.NewRunObservatoriumCommand),
		di.Provide(errors.NewErrorsCommand),
		di.Provide(environments2.Func(ServiceProviders)),
		di.Provide(migrations.New),

		metrics.ConfigProviders(),
	)
}

// ServiceProviders ...
func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(services.NewClusterService),
		di.Provide(services.NewDinosaurService, di.As(new(services.DinosaurService))),
		di.Provide(services.NewCloudProvidersService),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewFleetshardOperatorAddon),
		di.Provide(services.NewClusterPlacementStrategy),
		di.Provide(services.NewDataPlaneClusterService, di.As(new(services.DataPlaneClusterService))),
		di.Provide(services.NewDataPlaneDinosaurService, di.As(new(services.DataPlaneDinosaurService))),
		di.Provide(handlers.NewAuthenticationBuilder),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(routes.NewRouteLoader),
		di.Provide(quota.NewDefaultQuotaServiceFactory),
		di.Provide(workers.NewClusterManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewAcceptedDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewPreparingDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewDeletingDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewProvisioningDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewReadyDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaur_mgrs.NewDinosaurCNAMEManager, di.As(new(workers.Worker))),
	)
}
