// Package config for email sender service
package config

import (
	"github.com/caarlos0/env/v6"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// Config contains this application's runtime configuration.
type Config struct {
	ClusterID     string `env:"CLUSTER_ID"`
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:":8080"`
	EnableHTTPS   bool   `env:"ENABLE_HTTPS" envDefault:"false"`
	HTTPSCertFile string `env:"HTTPS_CERT_FILE" envDefault:""`
	HTTPSKeyFile  string `env:"HTTPS_KEY_FILE" envDefault:""`
  MetricsAddress string        `env:"METRICS_ADDRESS" envDefault:":9090"`
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

	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "invalid configuration settings")
	}
	return &c, nil
}
