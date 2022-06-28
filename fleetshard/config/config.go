package config

import (
	"github.com/stackrox/rox/pkg/sync"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

var (
	once   sync.Once
	cfg    *Config
	cfgErr error
)

// Config contains this application's runtime configuration.
type Config struct {
	FleetManagerEndpoint string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	ClusterID            string        `env:"CLUSTER_ID"`
	RuntimePollPeriod    time.Duration `env:"RUNTIME_POLL_PERIOD" envDefault:"5s"`
	AuthType             string        `env:"AUTH_TYPE" envDefault:"OCM"`
	RHSSOTokenFilePath   string        `env:"RHSSO_TOKEN_FILE" envDefault:"/run/secrets/rhsso-token/token"`
	OCMRefreshToken      string        `env:"OCM_TOKEN"`
}

func loadConfig() {
	c := Config{}
	if err := env.Parse(&c); err != nil {
		cfgErr = errors.Wrapf(err, "Unable to parse runtime configuration from environment")
		return
	}
	if c.ClusterID == "" {
		cfgErr = errors.New("CLUSTER_ID unset in the environment")
		return
	}
	if c.FleetManagerEndpoint == "" {
		cfgErr = errors.New("FLEET_MANAGER_ENDPOINT unset in the environment")
		return
	}
	if c.AuthType == "" {
		cfgErr = errors.New("AUTH_TYPE unset in the environment")
		return
	}
	cfg = &c
}

// Singleton retrieves the current runtime configuration from the environment and returns it.
func Singleton() (*Config, error) {
	once.Do(loadConfig)
	return cfg, cfgErr
}
