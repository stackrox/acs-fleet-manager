package server

import (
	"github.com/spf13/pflag"
)

type MetricsConfig struct {
	BindAddress string `json:"bind_address"`
	EnableHTTPS bool   `json:"enable_https"`
}

func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		BindAddress: "localhost:8080",
		EnableHTTPS: false,
	}
}

func (s *MetricsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "metrics-server-bindaddress", s.BindAddress, "Metrics server bind adddress")
	fs.BoolVar(&s.EnableHTTPS, "enable-metrics-https", s.EnableHTTPS, "Enable HTTPS for metrics server")
}

func (s *MetricsConfig) ReadFiles() error {
	return nil
}
