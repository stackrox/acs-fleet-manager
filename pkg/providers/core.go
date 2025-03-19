// Package providers ...
package providers

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/server/profiler"
	"github.com/stackrox/acs-fleet-manager/pkg/serviceregistration"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sentry"
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
		di.Provide(ocm.NewAddonConfig, di.As(new(environments.ConfigModule))),
		di.Provide(iam.NewIAMConfig, di.As(new(environments.ConfigModule))),
		di.Provide(acl.NewAccessControlListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(quotamanagement.NewQuotaManagementListConfig, di.As(new(environments.ConfigModule))),
		di.Provide(server.NewMetricsConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewContextConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewFleetShardAuthZConfig, di.As(new(environments.ConfigModule))),
		di.Provide(auth.NewAdminAuthZConfig, di.As(new(environments.ConfigModule))),
		di.Provide(telemetry.NewTelemetryConfig, di.As(new(environments.ConfigModule))),

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

		di.Provide(func(config *ocm.OCMConfig) ocm.ClusterManagementClient {
			if config.EnableMock {
				return ocm.NewMockClient()
			}

			conn, _, err := ocm.NewOCMConnection(config, config.BaseURL)
			if err != nil {
				logger.Logger.Error(err)
			}
			return ocm.NewClient(conn)
		}),

		di.Provide(func(config *ocm.OCMConfig) ocm.AMSClient {
			if config.EnableMock {
				return ocm.NewMockClient()
			}

			conn, _, err := ocm.NewOCMConnection(config, config.AmsURL)
			if err != nil {
				logger.Logger.Error(err)
			}
			return ocm.NewClient(conn)
		}),

		di.Provide(aws.NewDefaultClientFactory, di.As(new(aws.ClientFactory))),

		di.Provide(acl.NewAccessControlListMiddleware),
		di.Provide(services.NewTelemetryAuth),

		// Types registered as a BootService are started when the env is started
		di.Provide(server.NewAPIServer, di.As(new(environments.BootService))),
		di.Provide(server.NewMetricsServer, di.As(new(environments.BootService))),
		di.Provide(profiler.SingletonPprofServer, di.As(new(environments.BootService))),
		di.Provide(server.NewHealthCheckServer, di.As(new(environments.BootService))),
		di.Provide(serviceregistration.NewService, di.As(new(environments.BootService))),
		di.Provide(services.NewTelemetry, di.As(new(environments.BootService))),
	)
}
