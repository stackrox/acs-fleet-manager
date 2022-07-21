package sentry

import (
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	"github.com/spf13/pflag"
)

// Config ...
type Config struct {
	Enabled bool          `json:"enabled"`
	Key     string        `json:"key"`
	URL     string        `json:"url"`
	Project string        `json:"project"`
	Debug   bool          `json:"debug"`
	Timeout time.Duration `json:"timeout"`

	KeyFile string `json:"key_file"`
}

// NewConfig ...
func NewConfig() *Config {
	return &Config{
		Enabled: false,
		Key:     "",
		URL:     "sentry.autom8.in",
		Project: "8", // 8 is the dev project, this might change
		Debug:   false,
		KeyFile: "secrets/sentry.key",
	}
}

// AddFlags ...
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.Enabled, "enable-sentry", c.Enabled, "Enable sentry error monitoring")
	fs.StringVar(&c.KeyFile, "sentry-key-file", c.KeyFile, "File containing Sentry key")
	fs.StringVar(&c.URL, "sentry-url", c.URL, "Base URL of Sentry isntance")
	fs.StringVar(&c.Project, "sentry-project", c.Project, "Sentry project to report to")
	fs.BoolVar(&c.Debug, "enable-sentry-debug", c.Debug, "Enable sentry error monitoring")
	fs.DurationVar(&c.Timeout, "sentry-timeout", c.Timeout, "Timeout for all requests made to Sentry")
}

// ReadFiles ...
func (c *Config) ReadFiles() error {
	return shared.ReadFileValueString(c.KeyFile, &c.Key)
}
