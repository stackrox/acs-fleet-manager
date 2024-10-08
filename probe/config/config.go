// Package config ...
package config

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"sigs.k8s.io/yaml"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

// Config contains this application's runtime configuration.
type Config struct {
	AuthType                string        `env:"AUTH_TYPE" envDefault:"RHSSO"`
	CentralSpecs            CentralSpecs  `env:"CENTRAL_SPECS" envDefault:"[{ \"cloudProvider\": \"standalone\", \"region\": \"standalone\" }]"`
	FleetManagerEndpoint    string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	MetricsAddress          string        `env:"METRICS_ADDRESS" envDefault:":7070"`
	RHSSOClientID           string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	ProbeName               string        `env:"PROBE_NAME" envDefault:"${HOSTNAME}" envExpand:"true"`
	ProbeCleanUpTimeout     time.Duration `env:"PROBE_CLEANUP_TIMEOUT" envDefault:"5m"`
	ProbeHTTPRequestTimeout time.Duration `env:"PROBE_HTTP_REQUEST_TIMEOUT" envDefault:"5s"`
	ProbePollPeriod         time.Duration `env:"PROBE_POLL_PERIOD" envDefault:"5s"`
	ProbeRunTimeout         time.Duration `env:"PROBE_RUN_TIMEOUT" envDefault:"30m"`
	ProbeRunWaitPeriod      time.Duration `env:"PROBE_RUN_WAIT_PERIOD" envDefault:"30s"`

	ProbeUsername string
}

// CentralSpecs container for the CentralSpec slice
type CentralSpecs []CentralSpec

// CentralSpec the desired central configuration
type CentralSpec struct {
	CloudProvider string `json:"cloudProvider"`
	Region        string `json:"region"`
}

// UnmarshalText implements encoding.TextUnmarshaler so that CentralSpec can be parsed by env.Parse
func (s *CentralSpecs) UnmarshalText(text []byte) error {
	var specs []CentralSpec
	if err := yaml.Unmarshal(text, &specs); err != nil {
		return fmt.Errorf("unmarshal central spec: %w", err)
	}
	*s = specs
	return nil
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
	default:
		configErrors.AddError(errors.New("AUTH_TYPE not supported"))
	}
	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}
