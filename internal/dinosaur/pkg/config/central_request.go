package config

import (
	"time"

	"github.com/spf13/pflag"
)

// CentralRequestConfig holds all configuration for CentralRequests, e.g. expiration timeouts.
type CentralRequestConfig struct {
	CentralRequestExpirationTimeout  time.Duration `json:"central_request_expiration_timeout"`
	CentralRequestInternalUserAgents []string      `json:"central_request_internal_user_agents"`
}

// NewCentralRequestConfig creates a new CentralRequestConfig with default values.
func NewCentralRequestConfig() *CentralRequestConfig {
	return &CentralRequestConfig{
		CentralRequestExpirationTimeout:  60 * time.Minute,
		CentralRequestInternalUserAgents: []string{"fleet-manager-probe-service"},
	}
}

// AddFlags adds flags for all configuration settings within CentralRequestConfig to the flag set.
func (c *CentralRequestConfig) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&c.CentralRequestExpirationTimeout, "central-request-expiration-timeout",
		c.CentralRequestExpirationTimeout, "Timeout for central requests")
	fs.StringSliceVar(&c.CentralRequestInternalUserAgents, "central-request-internal-user-agents",
		c.CentralRequestInternalUserAgents,
		"HTTP User-Agents for central requests coming from internal services such as the probe service")
}

// ReadFiles will read any files specified via flags.
// Note: this is required to satisfy the environment.ConfigModule interface and will be a no-op for this struct.
func (c *CentralRequestConfig) ReadFiles() error {
	return nil
}
