// Package telemetry holds telemetry configuration to be applied by dependency injection.
package telemetry

import (
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

// Telemeter is a wrapper interface for the telemeter interface to enable mock testing.
//
//go:generate moq -out telemeter_moq.go . Telemeter
type Telemeter interface {
	telemeter.Telemeter
}

// TelemetryConfig is a wrapper for the telemetry configuration.
//
//go:generate moq -out config_moq.go . TelemetryConfig
type TelemetryConfig interface {
	Enabled() bool
	Telemeter() telemeter.Telemeter

	AddFlags(fs *pflag.FlagSet)
	ReadFiles() error
}

// TelemetryConfigImpl is the default telemetry config implementation.
type TelemetryConfigImpl struct {
	phonehome.Config

	StorageKeyFile string
}

var _ TelemetryConfig = &TelemetryConfigImpl{}

// NewTelemetryConfig creates a new telemetry configuration.
func NewTelemetryConfig() TelemetryConfig {
	// HOSTNAME is set to the pod name by K8s.
	clientID := getEnv("HOSTNAME", "fleet-manager")
	return &TelemetryConfigImpl{
		Config: phonehome.Config{
			ClientID:     clientID,
			ClientName:   "ACS Fleet Manager",
			PushInterval: 1 * time.Minute,
		},
		StorageKeyFile: "secrets/telemetry.storageKey",
	}
}

// AddFlags adds telemetry CLI flags.
func (t *TelemetryConfigImpl) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&t.Endpoint, "telemetry-endpoint", t.Endpoint, "The telemetry endpoint")
	fs.StringVar(&t.StorageKeyFile, "telemetry-storage-key-secret-file", t.StorageKeyFile, "File containing the telemetry storage key")
}

// ReadFiles reads telemetry secret files.
func (t *TelemetryConfigImpl) ReadFiles() error {
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
