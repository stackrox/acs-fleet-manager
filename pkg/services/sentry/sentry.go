package sentry

import (
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// Initialize ...
func Initialize(envName environments.EnvName, c *Config) error {
	options := sentry.ClientOptions{}

	if c.Enabled {
		key := c.Key
		url := c.URL
		project := c.Project
		glog.Infof("Sentry error reporting enabled to %s on project %s", url, project)
		options.Dsn = fmt.Sprintf("https://%s@%s/%s", key, url, project)
	} else {
		// Setting the DSN to an empty string effectively disables sentry
		// See https://godoc.org/github.com/getsentry/sentry-go#ClientOptions Dsn
		glog.Infof("Disabling Sentry error reporting")
		options.Dsn = ""
	}

	options.Transport = &sentry.HTTPTransport{
		Timeout: c.Timeout,
	}
	options.Debug = c.Debug
	options.AttachStacktrace = true
	options.Environment = string(envName)

	hostname, err := os.Hostname()
	if err != nil && hostname != "" {
		options.ServerName = hostname
	}
	// TODO figure out some way to set options.Release and options.Dist

	err = sentry.Init(options)
	if err != nil {
		glog.Errorf("Unable to initialize sentry integration: %s", err.Error())
		return err
	}
	return nil
}
