package providers

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/cmd/migrate"
	"github.com/stackrox/acs-fleet-manager/pkg/cmd/serve"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sentry"
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
		di.Provide(server.NewHealthCheckConfig, di.As(new(environments.ConfigModule))),
		di.Provide(db.NewDatabaseConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.NewServerConfig, di.As(new(environments.ConfigModule))),
		di.Provide(ocm.NewOCMConfig, di.As(new(environments.ConfigModule))),
		di.Provide(iam.NewIAMConfig, di.As(new(environments.ConfigModule))),
		di.Provide(acl.NewAccessControlListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(quotamanagement.NewQuotaManagementListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.NewMetricsConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewContextConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewFleetShardAuthZConfig, di.As(new(environments.ConfigModule))),

		// Add common CLI sub commands
		di.Provide(serve.NewServeCommand),
		di.Provide(migrate.NewMigrateCommand),

		// Add other core config providers..
		sentry.ConfigProviders(),
		authorization.ConfigProviders(),
		account.ConfigProviders(),

		di.Provide(environments.Func(ServiceProviders)),
	)
}

// ServiceProviders ...
func ServiceProviders() di.Option {
	return di.Options(

		// provide the service constructors
		di.Provide(db.NewConnectionFactory),
		di.Provide(observatorium.NewObservatoriumClient),

		di.Provide(func(config *ocm.OCMConfig) ocm.ClusterManagementClient {
			conn, _, err := ocm.NewOCMConnection(config, config.BaseURL)
			if err != nil {
				logger.Logger.Error(err)
			}
			return ocm.NewClient(conn)
		}),

		di.Provide(func(config *ocm.OCMConfig) ocm.AMSClient {
			conn, _, err := ocm.NewOCMConnection(config, config.AmsURL)
			if err != nil {
				logger.Logger.Error(err)
			}
			return ocm.NewClient(conn)
		}),

		di.Provide(aws.NewDefaultClientFactory, di.As(new(aws.ClientFactory))),

		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(handlers.NewErrorsHandler),
		di.Provide(func(c *iam.IAMConfig) sso.IAMService {
			return sso.NewIAMService(c)
		}),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(environments.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(environments.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(environments.BootService))),
		di.Provide(workers.NewLeaderElectionManager, di.As(new(environments.BootService))),
	)
}
