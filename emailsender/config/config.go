// Package config for email sender service
package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// Config contains this application's runtime configuration.
type Config struct {
	ClusterID      string `env:"CLUSTER_ID"`
	ServerAddress  string `env:"SERVER_ADDRESS" envDefault:":8080"`
	EnableHTTPS    bool   `env:"ENABLE_HTTPS" envDefault:"false"`
	HTTPSCertFile  string `env:"HTTPS_CERT_FILE" envDefault:""`
	HTTPSKeyFile   string `env:"HTTPS_KEY_FILE" envDefault:""`
	MetricsAddress string `env:"METRICS_ADDRESS" envDefault:":9090"`
	AuthConfigFile string `env:"AUTH_CONFIG_FILE" envDefault:"config/emailsender-authz.yaml"`
	AuthConfig     AuthConfig
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	c := Config{}
	var configErrors errorhelpers.ErrorList

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrap(err, "unable to parse runtime configuration from environment")
	}

	if c.ClusterID == "" {
		configErrors.AddError(errors.New("CLUSTER_ID environment variable is not set"))
	}

	if c.EnableHTTPS {
		if c.HTTPSCertFile == "" || c.HTTPSKeyFile == "" {
			configErrors.AddError(errors.New("ENABLE_HTTPS is true but required variables HTTPS_CERT_FILE or HTTPS_KEY_FILE are empty"))
		}
	}

	auth := &AuthConfig{file: c.AuthConfigFile}
	if err := auth.ReadFile(); err != nil {
		configErrors.AddError(err)
	}

	c.AuthConfig = *auth

	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "invalid configuration settings")
	}
	return &c, nil
}

// AuthConfig is the configuration for authn/authz for the emailsender
type AuthConfig struct {
	file             string
	JwksURLs         []string `yaml:"jwks_urls"`
	AllowedIssuer    []string `yaml:"allowed_issuers"`
	AllowedOrgIDs    []string `yaml:"allowed_org_ids"`
	AllowedAudiences []string `yaml:"allowed_audiences"`
}

// ReadFile reads the config
func (c *AuthConfig) ReadFile() error {
	fileContents, err := shared.ReadFile(c.file)
	if err != nil {
		return fmt.Errorf("failed to read emailsender authz config: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal emailsender authz config: %w", err)
	}

	return nil
}
