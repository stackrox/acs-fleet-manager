// Package config ...
package config

import (
	"time"

	"github.com/caarlos0/env/v6"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// Config contains this application's runtime configuration.
type Config struct {
	StartupTimeout time.Duration `env:"STARTUP_TIMEOUT" envDefault:"300s"`
	ServerAddress  string        `env:"EMAIL_SENDER_SERVER_ADDRESS" envDefault:":8080"`
	MetricsAddress string        `env:"EMAIL_SENDER_METRICS_ADDRESS" envDefault:":9090"`
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	c := Config{}

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrap(err, "unable to parse runtime configuration from environment")
	}

	var configErrors errorhelpers.ErrorList
	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}
