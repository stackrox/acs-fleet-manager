package config

import (
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

// Config contains this application's runtime configuration.
type Config struct {
	FleetManagerEndpoint string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	ClusterID            string        `env:"CLUSTER_ID"`
	RuntimePollPeriod    time.Duration `env:"RUNTIME_POLL_PERIOD" envDefault:"1s"`
}

// Load retrieves the current runtime configuration from the environment and returns it.
func Load() (*Config, error) {
	c := Config{}
	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrapf(err, "Unable to parse runtime configuration from environment")
	}
	if c.ClusterID == "" {
		return nil, errors.New("CLUSTER_ID unset in the environment")
	}
	if c.FleetManagerEndpoint == "" {
		return nil, errors.New("FLEET_MANAGER_ENDPOINT unset in the environment")
	}

	return &c, nil
}
