// Package dinosaur ...
package dinosaur

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/migrations"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/routes"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services/quota"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers/dinosaurmgrs"
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
		di.Provide(config.NewCentralRequestConfig, di.As(new(environments2.ConfigModule))),

		di.Provide(environments2.Func(ServiceProviders)),
		di.Provide(migrations.New),
	)
}

// ServiceProviders ...
func ServiceProviders() di.Option {
	return di.Options(
		di.Provide(services.NewClusterService),
		di.Provide(services.NewDinosaurService),
		di.Provide(services.NewCloudProvidersService),
		di.Provide(services.NewObservatoriumService),
		di.Provide(services.NewClusterPlacementStrategy),
		di.Provide(services.NewDataPlaneClusterService),
		di.Provide(services.NewDataPlaneCentralService),
		di.Provide(handlers.NewAuthenticationBuilder),
		di.Provide(clusters.NewDefaultProviderFactory, di.As(new(clusters.ProviderFactory))),
		di.Provide(routes.NewRouteLoader),
		di.Provide(quota.NewDefaultQuotaServiceFactory),
		di.Provide(workers.NewClusterManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewAcceptedCentralManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewPreparingDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewDeletingDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewProvisioningDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewReadyDinosaurManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewDinosaurCNAMEManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewCentralAuthConfigManager, di.As(new(workers.Worker))),
		di.Provide(dinosaurmgrs.NewExpirationDateManager, di.As(new(workers.Worker))),
		di.Provide(gitops.NewEmptyReader),
		di.Provide(gitops.NewProvider),
		di.Provide(presenters.NewManagedCentralPresenter),
	)
}
