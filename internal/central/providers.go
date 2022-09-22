package central

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/central"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/cluster"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/cmd/observatorium"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/migrations"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/routes"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services/quota"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/workers/centralmgrs"
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
		di.Provide(config.NewCentralConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewDataplaneClusterConfig, di.As(new(environments2.ConfigModule))),
		di.Provide(config.NewFleetshardConfig, di.As(new(environments2.ConfigModule))),

		// Additional CLI subcommands
		di.Provide(cluster.NewClusterCommand),
		di.Provide(central.NewCentralCommand),
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
		di.Provide(services.NewCentralService, di.As(new(services.CentralService))),
		di.Provide(services.NewCloudProvidersService),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewFleetshardOperatorAddon),
		di.Provide(services.NewClusterPlacementStrategy),
		di.Provide(services.NewDataPlaneClusterService, di.As(new(services.DataPlaneClusterService))),
		di.Provide(services.NewDataPlaneCentralService, di.As(new(services.DataPlaneCentralService))),
		di.Provide(handlers.NewAuthenticationBuilder),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(routes.NewRouteLoader),
		di.Provide(quota.NewDefaultQuotaServiceFactory),
		di.Provide(workers.NewClusterManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewCentralManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewAcceptedCentralManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewPreparingCentralManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewDeletingCentralManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewProvisioningCentralManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewReadyCentralManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewCentralCNAMEManager, di.As(new(workers.Worker))),
		di.Provide(centralmgrs.NewCentralAuthConfigManager, di.As(new(workers.Worker))),
		di.Provide(presenters.NewManagedCentralPresenter),
	)
}
