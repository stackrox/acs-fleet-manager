package observatorium

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
)

type ObservabilityConfiguration struct {
	// Red Hat SSO configuration
	RedHatSsoGatewayUrl        string `json:"redhat_sso_gateway_url" yaml:"redhat_sso_gateway_url"`
	RedHatSsoAuthServerUrl     string `json:"redhat_sso_auth_server_url" yaml:"redhat_sso_auth_server_url"`
	RedHatSsoRealm             string `json:"redhat_sso_realm" yaml:"redhat_sso_realm"`
	RedHatSsoTenant            string `json:"redhat_sso_tenant" yaml:"redhat_sso_tenant"`
	RedHatSsoTokenRefresherUrl string `json:"redhat_sso_token_refresher_url" yaml:"redhat_sso_token_refresher_url"`
	MetricsClientId            string `json:"redhat_sso_metrics_client_id" yaml:"redhat_sso_metrics_client_id"`
	MetricsClientIdFile        string `json:"redhat_sso_metrics_client_id_file" yaml:"redhat_sso_metrics_client_id_file"`
	MetricsSecret              string `json:"redhat_sso_metrics_secret" yaml:"redhat_sso_metrics_secret"`
	MetricsSecretFile          string `json:"redhat_sso_metrics_secret_file" yaml:"redhat_sso_metrics_secret_file"`
	LogsClientId               string `json:"redhat_sso_logs_client_id" yaml:"redhat_sso_logs_client_id"`
	LogsClientIdFile           string `json:"redhat_sso_logs_client_id_file" yaml:"redhat_sso_logs_client_id_file"`
	LogsSecret                 string `json:"redhat_sso_logs_secret" yaml:"redhat_sso_logs_secret"`
	LogsSecretFile             string `json:"redhat_sso_logs_secret_file" yaml:"redhat_sso_logs_secret_file"`

	// Observatorium configuration
	Timeout    time.Duration `json:"timeout"`
	Insecure   bool          `json:"insecure"`
	Debug      bool          `json:"debug"`
	EnableMock bool          `json:"enable_mock"`

	// Configuration repo for the Observability operator
	ObservabilityConfigTag             string `json:"observability_config_tag"`
	ObservabilityConfigRepo            string `json:"observability_config_repo"`
	ObservabilityConfigChannel         string `json:"observability_config_channel"`
	ObservabilityConfigAccessToken     string `json:"observability_config_access_token"`
	ObservabilityConfigAccessTokenFile string `json:"observability_config_access_token_file"`
}

func NewObservabilityConfigurationConfig() *ObservabilityConfiguration {
	return &ObservabilityConfiguration{
		Timeout:                            240 * time.Second,
		Debug:                              true, // TODO: false
		EnableMock:                         false,
		Insecure:                           true, // TODO: false
		ObservabilityConfigRepo:            "https://api.github.com/repos/bf2fc6cc711aee1a0c2a/observability-resources-mk/contents",
		ObservabilityConfigChannel:         "resources", // Pointing to resources as the individual directories for prod and staging are no longer needed
		ObservabilityConfigAccessToken:     "",
		ObservabilityConfigAccessTokenFile: "secrets/observability-config-access.token",
		ObservabilityConfigTag:             "main",
		MetricsClientIdFile:                "secrets/rhsso-metrics.clientId",
		MetricsSecretFile:                  "secrets/rhsso-metrics.clientSecret",
		LogsClientIdFile:                   "secrets/rhsso-logs.clientId",
		LogsSecretFile:                     "secrets/rhsso-logs.clientSecret",
		RedHatSsoTenant:                    "",
		RedHatSsoAuthServerUrl:             "",
		RedHatSsoRealm:                     "",
		RedHatSsoTokenRefresherUrl:         "",
		RedHatSsoGatewayUrl:                "",
	}
}

func (c *ObservabilityConfiguration) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.RedHatSsoTenant, "observability-red-hat-sso-tenant", c.RedHatSsoTenant, "Red Hat SSO tenant")
	fs.StringVar(&c.RedHatSsoAuthServerUrl, "observability-red-hat-sso-auth-server-url", c.RedHatSsoAuthServerUrl, "Red Hat SSO auth server URL")
	fs.StringVar(&c.RedHatSsoGatewayUrl, "observability-red-hat-sso-observatorium-gateway", c.RedHatSsoGatewayUrl, "Red Hat SSO gateway URL")
	fs.StringVar(&c.RedHatSsoTokenRefresherUrl, "observability-red-hat-sso-token-refresher-url", c.RedHatSsoTokenRefresherUrl, "Red Hat SSO token refresher URL")
	fs.StringVar(&c.LogsClientIdFile, "observability-red-hat-sso-logs-client-id-file", c.LogsClientIdFile, "Red Hat SSO logs client id file")
	fs.StringVar(&c.MetricsClientIdFile, "observability-red-hat-sso-metrics-client-id-file", c.MetricsClientIdFile, "Red Hat SSO metrics client id file")
	fs.StringVar(&c.LogsSecretFile, "observability-red-hat-sso-logs-secret-file", c.LogsSecretFile, "Red Hat SSO logs secret file")
	fs.StringVar(&c.MetricsSecretFile, "observability-red-hat-sso-metrics-secret-file", c.MetricsSecretFile, "Red Hat SSO metrics secret file")
	fs.StringVar(&c.RedHatSsoRealm, "observability-red-hat-sso-realm", c.RedHatSsoRealm, "Red Hat SSO realm")

	fs.DurationVar(&c.Timeout, "observatorium-timeout", c.Timeout, "Timeout for Observatorium client")
	fs.BoolVar(&c.Insecure, "observatorium-ignore-ssl", c.Insecure, "ignore SSL Observatorium certificate")
	fs.BoolVar(&c.EnableMock, "enable-observatorium-mock", c.EnableMock, "Enable mock Observatorium client")
	fs.BoolVar(&c.Debug, "observatorium-debug", c.Debug, "Debug flag for Observatorium client")

	fs.StringVar(&c.ObservabilityConfigRepo, "observability-config-repo", c.ObservabilityConfigRepo, "Repo for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigChannel, "observability-config-channel", c.ObservabilityConfigChannel, "Channel for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigAccessTokenFile, "observability-config-access-token-file", c.ObservabilityConfigAccessTokenFile, "File contains the access token to the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigTag, "observability-config-tag", c.ObservabilityConfigTag, "Tag or branch to use inside the observability configuration repo")
}

func (c *ObservabilityConfiguration) ReadFiles() error {
	configFileError := c.ReadObservatoriumConfigFiles()
	if configFileError != nil {
		return configFileError
	}

	if c.ObservabilityConfigAccessToken == "" && c.ObservabilityConfigAccessTokenFile != "" {
		return shared.ReadFileValueString(c.ObservabilityConfigAccessTokenFile, &c.ObservabilityConfigAccessToken)
	}
	return nil
}

func (c *ObservabilityConfiguration) ReadObservatoriumConfigFiles() error {
	logsClientIdErr := shared.ReadFileValueString(c.LogsClientIdFile, &c.LogsClientId)
	if logsClientIdErr != nil {
		return logsClientIdErr
	}

	logsSecretErr := shared.ReadFileValueString(c.LogsSecretFile, &c.LogsSecret)
	if logsSecretErr != nil {
		return logsSecretErr
	}

	metricsClientIdErr := shared.ReadFileValueString(c.MetricsClientIdFile, &c.MetricsClientId)
	if metricsClientIdErr != nil {
		return metricsClientIdErr
	}

	metricsSecretErr := shared.ReadFileValueString(c.MetricsSecretFile, &c.MetricsSecret)
	if metricsSecretErr != nil {
		return metricsSecretErr
	}

	return nil
}
