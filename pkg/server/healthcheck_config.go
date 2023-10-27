package server

import (
	"github.com/spf13/pflag"
	"sync"
)

// HealthCheckConfig ...
type HealthCheckConfig struct {
	BindAddress string `json:"bind_address"`
	EnableHTTPS bool   `json:"enable_https"`
}

var (
	onceHealthCheckConfig sync.Once
	healthCheckConfig     *HealthCheckConfig
)

// GetHealthCheckConfig ...
func GetHealthCheckConfig() *HealthCheckConfig {
	onceHealthCheckConfig.Do(func() {
		healthCheckConfig = &HealthCheckConfig{
			BindAddress: "localhost:8083",
			EnableHTTPS: false,
		}
	})
	return healthCheckConfig
}

// AddFlags ...
func (c *HealthCheckConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.BindAddress, "health-check-server-bindaddress", c.BindAddress, "Health check server bind adddress")
	fs.BoolVar(&c.EnableHTTPS, "enable-health-check-https", c.EnableHTTPS, "Enable HTTPS for health check server")
}

// ReadFiles ...
func (c *HealthCheckConfig) ReadFiles() error {
	return nil
}
