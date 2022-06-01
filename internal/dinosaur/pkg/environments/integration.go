package environments

import (
	"os"

	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

type IntegrationEnvLoader struct{}

var _ environments.EnvLoader = IntegrationEnvLoader{}

func NewIntegrationEnvLoader() environments.EnvLoader {
	return IntegrationEnvLoader{}
}

func (b IntegrationEnvLoader) Defaults() map[string]string {
	return map[string]string{
		"v":                                    "0",
		"logtostderr":                          "true",
		"ocm-base-url":                         "https://api-integration.6943.hive-integration.openshiftapps.com",
		"ams-base-url":                         "https://api-integration.6943.hive-integration.openshiftapps.com",
		"enable-https":                         "false",
		"enable-metrics-https":                 "false",
		"enable-terms-acceptance":              "false",
		"ocm-debug":                            "false",
		"enable-ocm-mock":                      "true",
		"ocm-mock-mode":                        ocm.MockModeEmulateServer,
		"enable-sentry":                        "false",
		"enable-deny-list":                     "true",
		"enable-instance-limit-control":        "true",
		"max-allowed-instances":                "1",
		"sso-base-url":                         "https://sso.redhat.com",
		"osd-idp-sso-realm":                    "rhoas-dinosaur-sre",
		"enable-dinosaur-external-certificate": "false",
		"cluster-compute-machine-type":         "m5.xlarge",
		"allow-evaluator-instance":             "true",
		"quota-type":                           "quota-management-list",
		"enable-deletion-of-expired-dinosaur":  "true",
		"dataplane-cluster-scaling-type":       "auto", // need to set this to 'auto' for integration environment as some tests rely on this
		"dinosaur-operator-addon-id":           "managed-dinosaur-qe",
		"fleetshard-addon-id":                  "fleetshard-operator-qe",
	}
}

// The integration environment is specifically for automated integration testing using an emulated server
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func (b IntegrationEnvLoader) ModifyConfiguration(env *environments.Env) error {
	// Support a one-off env to allow enabling db debug in testing
	var databaseConfig *db.DatabaseConfig
	var observabilityConfiguration *observatorium.ObservabilityConfiguration
	env.MustResolveAll(&databaseConfig, &observabilityConfiguration)

	if os.Getenv("DB_DEBUG") == "true" {
		databaseConfig.Debug = true
	}
	observabilityConfiguration.EnableMock = true
	return nil
}
