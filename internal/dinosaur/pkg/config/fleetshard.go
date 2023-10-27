package config

import (
	"github.com/spf13/pflag"
	"sync"
)

// FleetshardConfig ...
type FleetshardConfig struct {
	PollInterval   string `json:"poll_interval"`
	ResyncInterval string `json:"resync_interval"`
}

var (
	onceFleetshardConfig     sync.Once
	fleetshardConfigInstance *FleetshardConfig
)

// GetFleetshardConfig returns the FleetshardConfig
func GetFleetshardConfig() *FleetshardConfig {
	onceFleetshardConfig.Do(func() {
		fleetshardConfigInstance = &FleetshardConfig{
			PollInterval:   "15s",
			ResyncInterval: "60s",
		}
	})
	return fleetshardConfigInstance
}

// AddFlags ...
func (c *FleetshardConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.PollInterval, "fleetshard-poll-interval", c.PollInterval, "Interval defining how often the synchronizer polls and gets updates from the control plane")
	fs.StringVar(&c.ResyncInterval, "fleetshard-resync-interval", c.ResyncInterval, "Interval defining how often the synchronizer reports back status changes to the control plane")
}

// ReadFiles ...
func (c *FleetshardConfig) ReadFiles() error {
	return nil
}
