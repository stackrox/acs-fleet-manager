package config

import "github.com/spf13/pflag"

type FleetshardConfig struct {
	PollInterval   string `json:"poll_interval"`
	ResyncInterval string `json:"resync_interval"`
}

func NewFleetshardConfig() *FleetshardConfig {
	return &FleetshardConfig{
		PollInterval:   "15s",
		ResyncInterval: "60s",
	}
}

func (c *FleetshardConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.PollInterval, "fleetshard-poll-interval", c.PollInterval, "Interval defining how often the synchronizer polls and gets updates from the control plane")
	fs.StringVar(&c.ResyncInterval, "fleetshard-resync-interval", c.ResyncInterval, "Interval defining how often the synchronizer reports back status changes to the control plane")
}

func (c *FleetshardConfig) ReadFiles() error {
	return nil
}
