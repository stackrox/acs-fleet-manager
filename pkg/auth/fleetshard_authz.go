package auth

import (
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
	"gopkg.in/yaml.v2"
)

type AllowedOrgIDs []string

func (allowedOrgIDs AllowedOrgIDs) IsOrgIDAllowed(orgID string) bool {
	return arrays.FindFirstString(allowedOrgIDs, func(allowedOrgID string) bool {
		return orgID == allowedOrgID
	}) != -1
}

type FleetShardAuthZConfig struct {
	Enabled           bool
	AllowedOrgIDs     AllowedOrgIDs
	AllowedOrgIDsFile string
}

func NewFleetShardAuthZConfig() *FleetShardAuthZConfig {
	return &FleetShardAuthZConfig{
		Enabled:           true,
		AllowedOrgIDsFile: "config/fleetshard-authz-org-ids-prod.yaml",
	}
}

func (c *FleetShardAuthZConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AllowedOrgIDsFile, "fleetshard-authz-config-file", c.AllowedOrgIDsFile,
		"Fleetshard authZ middleware configuration file containing a list of allowed org IDs")
	fs.BoolVar(&c.Enabled, "enable-fleetshard-authz", c.Enabled, "Enable fleetshard authZ "+
		"via the list of allowed org IDs")
}

func (c *FleetShardAuthZConfig) ReadFiles() error {
	if c.Enabled {
		return readFleetShardAuthZConfigFile(c.AllowedOrgIDsFile, &c.AllowedOrgIDs)
	}

	return nil
}

func readFleetShardAuthZConfigFile(file string, val *AllowedOrgIDs) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {

	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
