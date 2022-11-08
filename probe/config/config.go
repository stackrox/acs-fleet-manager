// Package config ...
package config

import (
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

// Config contains this application's runtime configuration.
type Config struct {
	DataCloudProvider       string        `env:"DATA_PLANE_CLOUD_PROVIDER" envDefault:"aws"`
	DataPlaneRegion         string        `env:"DATA_PLANE_REGION" envDefault:"us-east-1"`
	FleetManagerEndpoint    string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	MetricsAddress          string        `env:"METRICS_ADDRESS" envDefault:":7070"`
	RHSSOClientID           string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	RHSSOClientSecret       string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET"`
	RHSSOEndpoint           string        `env:"RHSSO_ENDPOINT" envDefault:"https://sso.redhat.com"`
	RHSSORealm              string        `env:"RHSSO_REALM" envDefault:"redhat-external"`
	ProbeName               string        `env:"PROBE_NAME" envDefault:"pod"`
	ProbeNamePrefix         string        `env:"PROBE_NAME_PREFIX" envDefault:"probe"`
	ProbeCleanUpTimeout     time.Duration `env:"PROBE_CLEANUP_TIMEOUT" envDefault:"15m"`
	ProbeHTTPRequestTimeout time.Duration `env:"PROBE_HTTP_REQUEST_TIMEOUT" envDefault:"5s"`
	ProbePollPeriod         time.Duration `env:"PROBE_POLL_PERIOD" envDefault:"5s"`
	ProbeRunTimeout         time.Duration `env:"PROBE_RUN_TIMEOUT" envDefault:"15m"`
	ProbeRunWaitPeriod      time.Duration `env:"PROBE_RUN_WAIT_PERIOD" envDefault:"30s"`
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	var c Config

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrap(err, "unable to parse runtime configuration from environment")
	}

	var configErrors errorhelpers.ErrorList
	if c.RHSSOClientID == "" {
		configErrors.AddError(errors.New("RHSSO_SERVICE_ACCOUNT_CLIENT_ID unset in the environment"))
	}
	if c.RHSSOClientSecret == "" {
		configErrors.AddError(errors.New("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET unset in the environment"))
	}
	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}
