package config

import (
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/client/telemetry"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sentry"
)

func GetConfigs() []environments.ConfigModule {
	return []environments.ConfigModule{
		ocm.GetOCMConfig(),
		server.GetServerConfig(),
		db.GetDatabaseConfig(),
		iam.GetIAMConfig(),
		server.GetHealthCheckConfig(),
		acl.GetAccessControlListConfig(),
		quotamanagement.GetQuotaManagementListConfig(),
		server.GetMetricsConfig(),
		auth.GetContextConfig(),
		auth.GetFleetShardAuthZConfig(),
		auth.GetAdminAuthZConfig(),
		telemetry.GetTelemetryConfig(),
		sentry.GetConfig(),

		GetAWSConfig(),
		GetProviderConfig(),
		observatorium.GetObservabilityConfigurationConfig(),
		GetCentralConfig(),
		GetDataplaneClusterConfig(),
		GetFleetshardConfig(),
		GetCentralRequestConfig(),
	}
}
