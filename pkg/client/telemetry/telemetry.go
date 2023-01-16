// Package telemetry ...
package telemetry

import (
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// TelemetryConfig is a wrapper for the telemetry configuration.
type TelemetryConfig struct {
	phonehome.Config
}

// NewTelemetryConfig creates a new telemetry configuration.
func NewTelemetryConfig() *TelemetryConfig {
	return &TelemetryConfig{}
}

// AddFlags adds telemetry CLI flags.
func (t *TelemetryConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&t.Endpoint, "telemetry-endpoint", t.Config.Endpoint, "The telemetry endpoint")
	fs.StringVar(&t.StorageKey, "telemetry-storage-key", t.Config.StorageKey, "The telemetry storage key")
}

// ReadFiles reads telemetry secret files.
func (t *TelemetryConfig) ReadFiles() error {
	return nil
}
