package auth

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
	"gopkg.in/yaml.v2"
)

// ClaimValues a list of claim values that a fleetshard access token may contain
type ClaimValues []string

// Contains returns true if the specified value is present in the list
func (v ClaimValues) Contains(value string) bool {
	return arrays.FindFirstString(v, func(allowedValue string) bool {
		return value == allowedValue
	}) != -1
}

// FleetShardAuthZConfig ...
type FleetShardAuthZConfig struct {
	Enabled          bool        `yaml:"-"`
	File             string      `yaml:"-"`
	AllowedSubjects  ClaimValues `yaml:"allowed_subjects"`
	AllowedAudiences ClaimValues `yaml:"allowed_audiences"`
}

// NewFleetShardAuthZConfig ...
func NewFleetShardAuthZConfig() *FleetShardAuthZConfig {
	return &FleetShardAuthZConfig{
		Enabled: true,
		File:    "config/fleetshard-authz-prod.yaml",
	}
}

// AddFlags ...
func (c *FleetShardAuthZConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.File, "fleetshard-authz-config-file", c.File,
		"Fleetshard authZ middleware configuration file containing a list of allowed org IDs")
	fs.BoolVar(&c.Enabled, "enable-fleetshard-authz", c.Enabled, "Enable fleetshard authZ "+
		"via the list of allowed org IDs")
}

// ReadFiles ...
func (c *FleetShardAuthZConfig) ReadFiles() error {
	if c.Enabled {
		return readFleetShardAuthZConfigFile(c.File, c)
	}

	return nil
}

func readFleetShardAuthZConfigFile(file string, config *FleetShardAuthZConfig) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading FleedShard AuthZ config: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), config)
	if err != nil {
		return fmt.Errorf("unmarshalling FleedShard AuthZ config: %w", err)
	}

	return nil
}
