package config

import (
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
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
	CreateAuthProvider   bool          `env:"CREATE_AUTH_PROVIDER" envDefault:"false"`
	MetricsAddress       string        `env:"FLEETSHARD_METRICS_ADDRESS" envDefault:":8080"`
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	c := Config{}
	var configErrors errorhelpers.ErrorList

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrapf(err, "Unable to parse runtime configuration from environment")
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
	cfgErr := configErrors.ToError()
	if cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}
