package config

import (
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

// Config contains this application's runtime configuration.
type Config struct {
	DataCloudProvider    string        `env:"DATA_PLANE_CLOUD_PROVIDER" envDefault:"aws"`
	DataPlaneRegion      string        `env:"DATA_PLANE_REGION" envDefault:"us-east-1"`
	FleetManagerEndpoint string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	MetricsAddress       string        `env:"FLEETSHARD_METRICS_ADDRESS" envDefault:":7070"`
	RHSSOClientID        string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	RHSSOClientSecret    string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET"`
	RHSSOEndpoint        string        `env:"RHSSO_ENDPOINT" envDefault:"https://sso.redhat.com"`
	RHSSORealm           string        `env:"RHSSO_REALM" envDefault:"redhat-external"`
	RuntimePollPeriod    time.Duration `env:"RUNTIME_POLL_PERIOD" envDefault:"5s"`
	RuntimePollTimeout   time.Duration `env:"RUNTIME_POLL_TIMEOUT" envDefault:"5m"`
	RuntimeRunTimeout    time.Duration `env:"RUNTIME_RUN_TIMEOUT" envDefault:"15m"`
	RuntimeRunWaitPeriod time.Duration `env:"RUNTIME_RUN_WAIT_PERIOD" envDefault:"30s"`
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	c := Config{}
	var configErrors errorhelpers.ErrorList

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrapf(err, "Unable to parse runtime configuration from environment.")
	}
	if c.RHSSOClientID == "" {
		configErrors.AddError(errors.New("RHSSO_SERVICE_ACCOUNT_CLIENT_ID unset in the environment"))
	}
	if c.RHSSOClientSecret == "" {
		configErrors.AddError(errors.New("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET unset in the environment"))
	}
	cfgErr := configErrors.ToError()
	if cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "Unexpected configuration settings.")
	}
	return &c, nil
}
