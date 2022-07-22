package observatorium

import (
	"time"

	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// ObservabilityConfiguration ...
type ObservabilityConfiguration struct {
	// Red Hat SSO configuration
	RedHatSSOGatewayURL        string `json:"redhat_sso_gateway_url" yaml:"redhat_sso_gateway_url"`
	RedHatSSOAuthServerURL     string `json:"redhat_sso_auth_server_url" yaml:"redhat_sso_auth_server_url"`
	RedHatSSORealm             string `json:"redhat_sso_realm" yaml:"redhat_sso_realm"`
	RedHatSSOTenant            string `json:"redhat_sso_tenant" yaml:"redhat_sso_tenant"`
	RedHatSSOTokenRefresherURL string `json:"redhat_sso_token_refresher_url" yaml:"redhat_sso_token_refresher_url"`
	MetricsClientID            string `json:"redhat_sso_metrics_client_id" yaml:"redhat_sso_metrics_client_id"`
	MetricsClientIDFile        string `json:"redhat_sso_metrics_client_id_file" yaml:"redhat_sso_metrics_client_id_file"`
	MetricsSecret              string `json:"redhat_sso_metrics_secret" yaml:"redhat_sso_metrics_secret"`
	MetricsSecretFile          string `json:"redhat_sso_metrics_secret_file" yaml:"redhat_sso_metrics_secret_file"`
	LogsClientID               string `json:"redhat_sso_logs_client_id" yaml:"redhat_sso_logs_client_id"`
	LogsClientIDFile           string `json:"redhat_sso_logs_client_id_file" yaml:"redhat_sso_logs_client_id_file"`
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

// NewObservabilityConfigurationConfig ...
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
		MetricsClientIDFile:                "secrets/rhsso-metrics.clientId",
		MetricsSecretFile:                  "secrets/rhsso-metrics.clientSecret",
		LogsClientIDFile:                   "secrets/rhsso-logs.clientId",
		LogsSecretFile:                     "secrets/rhsso-logs.clientSecret",
		RedHatSSOTenant:                    "",
		RedHatSSOAuthServerURL:             "",
		RedHatSSORealm:                     "",
		RedHatSSOTokenRefresherURL:         "",
		RedHatSSOGatewayURL:                "",
	}
}

// AddFlags ...
func (c *ObservabilityConfiguration) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.RedHatSSOTenant, "observability-red-hat-sso-tenant", c.RedHatSSOTenant, "Red Hat SSO tenant")
	fs.StringVar(&c.RedHatSSOAuthServerURL, "observability-red-hat-sso-auth-server-url", c.RedHatSSOAuthServerURL, "Red Hat SSO auth server URL")
	fs.StringVar(&c.RedHatSSOGatewayURL, "observability-red-hat-sso-observatorium-gateway", c.RedHatSSOGatewayURL, "Red Hat SSO gateway URL")
	fs.StringVar(&c.RedHatSSOTokenRefresherURL, "observability-red-hat-sso-token-refresher-url", c.RedHatSSOTokenRefresherURL, "Red Hat SSO token refresher URL")
	fs.StringVar(&c.LogsClientIDFile, "observability-red-hat-sso-logs-client-id-file", c.LogsClientIDFile, "Red Hat SSO logs client id file")
	fs.StringVar(&c.MetricsClientIDFile, "observability-red-hat-sso-metrics-client-id-file", c.MetricsClientIDFile, "Red Hat SSO metrics client id file")
	fs.StringVar(&c.LogsSecretFile, "observability-red-hat-sso-logs-secret-file", c.LogsSecretFile, "Red Hat SSO logs secret file")
	fs.StringVar(&c.MetricsSecretFile, "observability-red-hat-sso-metrics-secret-file", c.MetricsSecretFile, "Red Hat SSO metrics secret file")
	fs.StringVar(&c.RedHatSSORealm, "observability-red-hat-sso-realm", c.RedHatSSORealm, "Red Hat SSO realm")

	fs.DurationVar(&c.Timeout, "observatorium-timeout", c.Timeout, "Timeout for Observatorium client")
	fs.BoolVar(&c.Insecure, "observatorium-ignore-ssl", c.Insecure, "ignore SSL Observatorium certificate")
	fs.BoolVar(&c.EnableMock, "enable-observatorium-mock", c.EnableMock, "Enable mock Observatorium client")
	fs.BoolVar(&c.Debug, "observatorium-debug", c.Debug, "Debug flag for Observatorium client")

	fs.StringVar(&c.ObservabilityConfigRepo, "observability-config-repo", c.ObservabilityConfigRepo, "Repo for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigChannel, "observability-config-channel", c.ObservabilityConfigChannel, "Channel for the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigAccessTokenFile, "observability-config-access-token-file", c.ObservabilityConfigAccessTokenFile, "File contains the access token to the observability operator configuration repo")
	fs.StringVar(&c.ObservabilityConfigTag, "observability-config-tag", c.ObservabilityConfigTag, "Tag or branch to use inside the observability configuration repo")
}

// ReadFiles ...
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

// ReadObservatoriumConfigFiles ...
func (c *ObservabilityConfiguration) ReadObservatoriumConfigFiles() error {
	logsClientIDErr := shared.ReadFileValueString(c.LogsClientIDFile, &c.LogsClientID)
	if logsClientIDErr != nil {
		return logsClientIDErr
	}

	logsSecretErr := shared.ReadFileValueString(c.LogsSecretFile, &c.LogsSecret)
	if logsSecretErr != nil {
		return logsSecretErr
	}

	metricsClientIDErr := shared.ReadFileValueString(c.MetricsClientIDFile, &c.MetricsClientID)
	if metricsClientIDErr != nil {
		return metricsClientIDErr
	}

	metricsSecretErr := shared.ReadFileValueString(c.MetricsSecretFile, &c.MetricsSecret)
	if metricsSecretErr != nil {
		return metricsSecretErr
	}

	return nil
}
