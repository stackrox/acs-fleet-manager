package config

import (
	"github.com/stackrox/rox/pkg/errorhelpers"
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
	StaticToken          string        `env:"STATIC_TOKEN"`
}

func loadConfig() {
	c := Config{}
	var configErrors errorhelpers.ErrorList

	if err := env.Parse(&c); err != nil {
		cfgErr = errors.Wrapf(err, "Unable to parse runtime configuration from environment")
		return
	}
	if c.ClusterID == "" {
		configErrors.AddError(errors.New("CLUSTER_ID unset in the environment"))
	}
	if c.FleetManagerEndpoint == "" {
		configErrors.AddError(errors.New("FLEET_MANAGER_ENDPOINT unset in the environment"))
	}
	if c.AuthType == "" {
		configErrors.AddError(errors.New("AUTH_TYPE unset in the environment"))
	}
	cfgErr = configErrors.ToError()
	if cfgErr == nil {
		cfg = &c
	}
}

// Singleton retrieves the current runtime configuration from the environment and returns it.
func Singleton() (*Config, error) {
	once.Do(loadConfig)
	return cfg, cfgErr
}
