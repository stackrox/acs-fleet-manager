package server

import (
	"github.com/spf13/pflag"
)

// MetricsConfig ...
type MetricsConfig struct {
	BindAddress string `json:"bind_address"`
	EnableHTTPS bool   `json:"enable_https"`
}

// NewMetricsConfig ...
func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		BindAddress: "localhost:8080",
		EnableHTTPS: false,
	}
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
