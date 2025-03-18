// Package config ...
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// Config contains this application's runtime configuration.
type Config struct {
	AuthType                string        `env:"AUTH_TYPE" envDefault:"RHSSO"`
	FleetManagerEndpoint    string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	MetricsAddress          string        `env:"METRICS_ADDRESS" envDefault:":7070"`
	RHSSOClientID           string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	ProbeName               string        `env:"PROBE_NAME" envDefault:"${HOSTNAME}" envExpand:"true"`
	ProbeHTTPRequestTimeout time.Duration `env:"PROBE_HTTP_REQUEST_TIMEOUT" envDefault:"5s"`
	ProbePollPeriod         time.Duration `env:"PROBE_POLL_PERIOD" envDefault:"5s"`
	ProbeRunTimeout         time.Duration `env:"PROBE_RUN_TIMEOUT" envDefault:"35m"`
	ProbeRunWaitPeriod      time.Duration `env:"PROBE_RUN_WAIT_PERIOD" envDefault:"30s"`

	ProbeUsername string
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (Config, error) {
	// Default value if PROBE_NAME and HOSTNAME are not set.
	c := Config{ProbeName: "probe"}

	if err := env.Parse(&c); err != nil {
		return c, errors.Wrap(err, "unable to parse runtime configuration from environment")
	}

	var configErrors errorhelpers.ErrorList
	switch c.AuthType {
	case "RHSSO":
		if c.RHSSOClientID == "" {
			configErrors.AddError(errors.New("RHSSO_SERVICE_ACCOUNT_CLIENT_ID unset in the environment"))
		}
		c.ProbeUsername = fmt.Sprintf("service-account-%s", c.RHSSOClientID)
	default:
		configErrors.AddError(errors.New("AUTH_TYPE not supported"))
	}
	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return c, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return c, nil
}
