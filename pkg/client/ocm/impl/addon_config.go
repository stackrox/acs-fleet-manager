package impl

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// AddonConfig addon service configuration
type AddonConfig struct {
	URL                           string
	ClientID                      string
	ClientIDFile                  string
	ClientSecret                  string
	ClientSecretFile              string
	SelfToken                     string
	SelfTokenFile                 string
	InheritFleetshardSyncImageTag bool
	FleetshardSyncImageTag        string
}

// NewAddonConfig creates a new instance of AddonConfig
func NewAddonConfig() *AddonConfig {
	return &AddonConfig{
		URL:                           "https://api.openshift.com",
		ClientIDFile:                  "secrets/ocm-addon-service.clientId",
		ClientSecretFile:              "secrets/ocm-addon-service.clientSecret", // pragma: allowlist secret
		SelfTokenFile:                 "secrets/ocm-addon-service.token",
		InheritFleetshardSyncImageTag: true,
	}
}

// AddFlags ...
func (c *AddonConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ClientIDFile, "ocm-addon-client-id-file", c.ClientIDFile, "File containing OCM API privileged account client-id")
	fs.StringVar(&c.ClientSecretFile, "ocm-addon-client-secret-file", c.ClientSecretFile, "File containing OCM API privileged account client-secret")
	fs.StringVar(&c.SelfTokenFile, "addon-self-token-file", c.SelfTokenFile, "File containing OCM API privileged offline SSO token")
	fs.StringVar(&c.URL, "ocm-addon-url", c.URL, "The base URL of the OCM API, integration by default")
	fs.BoolVar(&c.InheritFleetshardSyncImageTag, "inherit-fleetshard-sync-image-tag", c.InheritFleetshardSyncImageTag, "Enable fleetshard-sync image tag")
	fs.StringVar(&c.FleetshardSyncImageTag, "fleetshard-sync-image-tag", c.FleetshardSyncImageTag, "Fleetshard-sync image tag")
}

// ReadFiles ...
func (c *AddonConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.ClientIDFile, &c.ClientID)
	if err != nil {
		return fmt.Errorf("reading client ID file: %w", err)
	}
	err = shared.ReadFileValueString(c.ClientSecretFile, &c.ClientSecret)
	if err != nil {
		return fmt.Errorf("reading client secret file: %w", err)
	}
	err = shared.ReadFileValueString(c.SelfTokenFile, &c.SelfToken)
	if err != nil && (c.ClientSecret == "" || c.ClientID == "") {
		return fmt.Errorf("reading self token file: %w", err)
	}

	return nil
}
