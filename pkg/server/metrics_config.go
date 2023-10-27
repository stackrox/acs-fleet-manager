package server

import (
	"github.com/spf13/pflag"
	"sync"
)

// MetricsConfig ...
type MetricsConfig struct {
	BindAddress string `json:"bind_address"`
	EnableHTTPS bool   `json:"enable_https"`
}

var (
	onceMetricsConfig sync.Once
	metricsConfig     *MetricsConfig
)

// GetMetricsConfig ...
func GetMetricsConfig() *MetricsConfig {
	onceMetricsConfig.Do(func() {
		metricsConfig = &MetricsConfig{
			BindAddress: "localhost:8080",
			EnableHTTPS: false,
		}
	})
	return metricsConfig
}

// AddFlags ...
func (s *MetricsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "metrics-server-bindaddress", s.BindAddress, "Metrics server bind adddress")
	fs.BoolVar(&s.EnableHTTPS, "enable-metrics-https", s.EnableHTTPS, "Enable HTTPS for metrics server")
}

// ReadFiles ...
func (s *MetricsConfig) ReadFiles() error {
	return nil
}
