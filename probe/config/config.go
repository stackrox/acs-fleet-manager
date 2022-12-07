// Package config ...
package config

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

// Config contains this application's runtime configuration.
type Config struct {
	AuthType                string        `env:"AUTH_TYPE" envDefault:"RHSSO"`
	DataCloudProvider       string        `env:"DATA_PLANE_CLOUD_PROVIDER" envDefault:"aws"`
	DataPlaneRegion         string        `env:"DATA_PLANE_REGION" envDefault:"us-east-1"`
	FleetManagerEndpoint    string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	MetricsAddress          string        `env:"METRICS_ADDRESS" envDefault:":7070"`
	RHSSOClientID           string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	OCMUsername             string        `env:"OCM_USERNAME"`
	ProbeName               string        `env:"PROBE_NAME" envDefault:"${HOSTNAME}" envExpand:"true"`
	ProbeCleanUpTimeout     time.Duration `env:"PROBE_CLEANUP_TIMEOUT" envDefault:"5m"`
	ProbeHTTPRequestTimeout time.Duration `env:"PROBE_HTTP_REQUEST_TIMEOUT" envDefault:"5s"`
	ProbePollPeriod         time.Duration `env:"PROBE_POLL_PERIOD" envDefault:"5s"`
	ProbeRunTimeout         time.Duration `env:"PROBE_RUN_TIMEOUT" envDefault:"30m"`
	ProbeRunWaitPeriod      time.Duration `env:"PROBE_RUN_WAIT_PERIOD" envDefault:"30s"`

	ProbeUsername string
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	// Default value if PROBE_NAME and HOSTNAME are not set.
	c := Config{ProbeName: "probe"}

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrap(err, "unable to parse runtime configuration from environment")
	}

	var configErrors errorhelpers.ErrorList
	switch c.AuthType {
	case "RHSSO":
		if c.RHSSOClientID == "" {
			configErrors.AddError(errors.New("RHSSO_SERVICE_ACCOUNT_CLIENT_ID unset in the environment"))
		}
		c.ProbeUsername = fmt.Sprintf("service-account-%s", c.RHSSOClientID)
	case "OCM":
		if c.OCMUsername == "" {
			configErrors.AddError(errors.New("OCM_USERNAME unset in the environment"))
		}
		c.ProbeUsername = c.OCMUsername
	default:
		configErrors.AddError(errors.New("AUTH_TYPE not supported"))
	}
	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}
