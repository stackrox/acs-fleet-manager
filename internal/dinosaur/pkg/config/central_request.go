package config

import (
	"sync"
	"time"

	"github.com/spf13/pflag"
)

var (
	onceCentralRequestConfig sync.Once
	centralRequestConfig     *CentralRequestConfig
)

// CentralRequestConfig holds all configuration for CentralRequests, e.g. expiration timeouts.
type CentralRequestConfig struct {
	ExpirationTimeout  time.Duration `json:"expiration_timeout"`
	InternalUserAgents []string      `json:"internal_user_agents"`
}

// GetCentralRequestConfig creates a new CentralRequestConfig with default values.
func GetCentralRequestConfig() *CentralRequestConfig {
	onceCentralRequestConfig.Do(func() {
		centralRequestConfig = &CentralRequestConfig{
			ExpirationTimeout:  60 * time.Minute,
			InternalUserAgents: []string{"fleet-manager-probe-service"},
		}
	})
	return centralRequestConfig
}

// AddFlags adds flags for all configuration settings within CentralRequestConfig to the flag set.
func (c *CentralRequestConfig) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&c.ExpirationTimeout, "central-request-expiration-timeout",
		c.ExpirationTimeout, "Timeout for central requests")
	fs.StringSliceVar(&c.InternalUserAgents, "central-request-internal-user-agents",
		c.InternalUserAgents,
		"HTTP User-Agents for central requests coming from internal services such as the probe service")
}

// ReadFiles will read any files specified via flags.
// Note: this is required to satisfy the environment.ConfigModule interface and will be a no-op for this struct.
func (c *CentralRequestConfig) ReadFiles() error {
	return nil
}
