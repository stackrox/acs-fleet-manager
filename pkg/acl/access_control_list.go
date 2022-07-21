package acl

import (
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
	"gopkg.in/yaml.v2"
)

// DeniedUsers ...
type DeniedUsers []string

// IsUserDenied ...
func (deniedAccounts DeniedUsers) IsUserDenied(username string) bool {
	return arrays.FindFirstString(deniedAccounts, func(user string) bool {
		return username == user
	}) != -1
}

// AccessControlListConfig ...
type AccessControlListConfig struct {
	DenyList           DeniedUsers
	DenyListConfigFile string
	EnableDenyList     bool
}

// NewAccessControlListConfig ...
func NewAccessControlListConfig() *AccessControlListConfig {
	return &AccessControlListConfig{
		DenyListConfigFile: "config/deny-list-configuration.yaml",
		EnableDenyList:     false,
	}
}

// AddFlags ...
func (c *AccessControlListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DenyListConfigFile, "deny-list-config-file", c.DenyListConfigFile, "DenyList configuration file")
	fs.BoolVar(&c.EnableDenyList, "enable-deny-list", c.EnableDenyList, "Enable access control via the denied list of users")
}

// ReadFiles ...
func (c *AccessControlListConfig) ReadFiles() (err error) {
	if c.EnableDenyList {
		return readDenyListConfigFile(c.DenyListConfigFile, &c.DenyList)
	}

	return nil
}

// Read the contents of file into the deny list config
func readDenyListConfigFile(file string, val *DeniedUsers) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
