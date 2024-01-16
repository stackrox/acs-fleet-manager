package ocm

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

// AddonConfig addon service configuration
type AddonConfig struct {
	URL                           string `env:"ADDON_SERVICE_OCM_URL" envDefault:"https://api.openshift.com"`
	ClientID                      string `env:"ADDON_SERVICE_OCM_CLIENT_ID_FILE,file" envDefault:"secrets/ocm-addon-service.clientId"`
	ClientSecret                  string `env:"ADDON_SERVICE_OCM_CLIENT_SECRET_FILE,file" envDefault:"secrets/ocm-addon-service.clientSecret"`
	SelfToken                     string `env:"ADDON_SERVICE_OCM_SELF_TOKEN_FILE,file" envDefault:"secrets/ocm-addon-service.token"`
	InheritFleetshardSyncImageTag bool   `env:"ADDON_SERVICE_INHERIT_FLEETSHARD_SYNC_IMAGE_TAG" envDefault:"true"`
	FleetshardSyncImageTag        string `env:"ADDON_SERVICE_FLEETSHARD_SYNC_IMAGE_TAG"`
}

// NewConfig creates a new instance of AddonConfig
func NewConfig() (*AddonConfig, error) {
	c := &AddonConfig{}
	if err := env.Parse(c); err != nil {
		return nil, fmt.Errorf("addon service config: %w", err)
	}
	// no validation here, because the ocm config is validated when a new connection is created (see ocm.NewOCMConnection)
	return c, nil
}
