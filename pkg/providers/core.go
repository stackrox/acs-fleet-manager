// Package providers ...
package providers

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// CoreConfigProviders ...
func CoreConfigProviders() di.Option {
	return di.Options(
		di.Provide(func(env *environments.Env) environments.EnvName {
			return environments.EnvName(env.Name)
		}),

		// Add config types
		di.Provide(server.GetHealthCheckConfig, di.As(new(environments.ConfigModule))),
		di.Provide(db.GetDatabaseConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.GetServerConfig, di.As(new(environments.ConfigModule))),
		di.Provide(ocm.GetOCMConfig, di.As(new(environments.ConfigModule))),
		di.Provide(iam.GetIAMConfig, di.As(new(environments.ConfigModule))),
		di.Provide(acl.GetAccessControlListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(quotamanagement.GetQuotaManagementListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.GetMetricsConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.GetContextConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.GetFleetShardAuthZConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.GetAdminAuthZConfig, di.As(new(environments.ConfigModule))),
		di.Provide(telemetry.GetTelemetryConfig, di.As(new(environments.ConfigModule))),
		di.Provide(config.GetCentralConfig, di.As(new(environments.ConfigModule))),
		di.Provide(config.GetProviderConfig, di.As(new(environments.ConfigModule))),

		// Add other core config providers..
		authorization.ConfigProviders(),
		account.ConfigProviders(),

		di.Provide(environments.Func(ServiceProviders)),
	)
}

// ServiceProviders ...
func ServiceProviders() di.Option {
	return di.Options(

		// provide the service constructors
		di.Provide(db.SingletonConnectionFactory),
		di.Provide(observatorium.SingletonObservatoriumClient),

		di.Provide(func() ocm.ClusterManagementClient {
			return ocm.SingletonClusterManagementClient()
		}),

		di.Provide(func() ocm.AMSClient {
			return ocm.SingletonAMSClient()
		}),

		di.Provide(aws.NewDefaultClientFactory, di.As(new(aws.ClientFactory))),

		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(handlers.NewErrorsHandler),
		di.Provide(func(c *iam.IAMConfig) sso.IAMService {
			return sso.SingletonIAMService()
		}),
		di.Provide(services.GetTelemetryAuth),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(environments.BootService))),
		di.Provide(server.SingletonMetricsServer, di.As(new(environments.BootService))),
		di.Provide(server.SingletonHealthCheckServer, di.As(new(environments.BootService))),
		di.Provide(workers.SingletonLeaderElectionManager, di.As(new(environments.BootService))),
		di.Provide(services.SingletonTelemetry, di.As(new(environments.BootService))),
		di.Provide(services.SingletonDataMigration, di.As(new(environments.BootService))),
	)
}
