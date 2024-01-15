package ocm

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/stackrox/rox/pkg/utils"
)

// AddonConfig addon service configuration
type AddonConfig struct {
	URL                           string `env:"ADDON_SERVICE_OCM_URL" envDefault:"https://api.openshift.com"`
	ClientID                      string `env:"ADDON_SERVICE_OCM_CLIENT_ID"`
	ClientSecret                  string `env:"ADDON_SERVICE_OCM_CLIENT_SECRET"`
	SelfToken                     string `env:"ADDON_SERVICE_OCM_SELF_TOKEN"`
	InheritFleetshardSyncImageTag bool   `env:"ADDON_SERVICE_INHERIT_FLEETSHARD_SYNC_IMAGE_TAG" envDefault:"true"`
	FleetshardSyncImageTag        string `env:"ADDON_SERVICE_FLEETSHARD_SYNC_IMAGE_TAG"`
}

// NewConfig creates a new instance of AddonConfig
func NewConfig() *AddonConfig {
	c := &AddonConfig{}
	if err := env.Parse(c); err != nil {
		utils.Should(fmt.Errorf("addon service config: %w", err))
	}
	// no validation here, because the ocm config is validated when a new connection is created (see ocm.NewOCMConnection)
	return c
}
