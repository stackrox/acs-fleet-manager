// Package telemetry holds telemetry configuration to be applied by dependency injection.
package telemetry

import (
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// TelemetryConfig is a wrapper for the telemetry configuration.
type TelemetryConfig struct {
	phonehome.Config

	StorageKeyFile string
}

// NewTelemetryConfig creates a new telemetry configuration.
func NewTelemetryConfig() *TelemetryConfig {
	// HOSTNAME is set to the pod name by K8s.
	clientID := getEnv("HOSTNAME", "fleet-manager")
	return &TelemetryConfig{
		Config: phonehome.Config{
			ClientID:   clientID,
			ClientName: "ACS Fleet Manager",
		},
		StorageKeyFile: "secrets/telemetry.storageKey",
	}
}

// AddFlags adds telemetry CLI flags.
func (t *TelemetryConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&t.Endpoint, "telemetry-endpoint", t.Endpoint, "The telemetry endpoint")
	fs.StringVar(&t.StorageKeyFile, "telemetry-storage-key-secret-file", t.StorageKeyFile, "File containing the telemetry storage key")
}

// ReadFiles reads telemetry secret files.
func (t *TelemetryConfig) ReadFiles() error {
	err := shared.ReadFileValueString(t.StorageKeyFile, &t.StorageKey)
	// Don't fail if telemetry secret key is not found.
	if err != nil {
		glog.Warningf("could not read telemetry storage key secret file")
	}
	return nil
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
