package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"gopkg.in/yaml.v2"
)

/*
# Could be a YAML as such:

# This file contains the role mapping for the admin API.
# Based on the HTTP method you can specify a list of roles authorized to do such a request.
# At the moment, GET PATCH DELETE (list / get, update, delete) are the three available methods by the admin API
- method: GET
  roles:
  - role-name
- method: PATCH
  roles:
  - role-name
- method: DELETE
  roles:
  - role-name
*/

// RolesConfiguration is the configuration of required roles per HTTP method of the admin API.
type RolesConfiguration struct {
	HTTPMethod string   `yaml:"method"`
	RoleNames  []string `yaml:"roles"`
}

// RoleAuthZConfig is the configuration of the role authZ middleware.
type RoleAuthZConfig struct {
	Enabled         bool
	RolesConfigFile string
	RolesConfig     []RolesConfiguration
}

// NewRoleAuthZConfig creates a default RoleAuthZConfig which is enabled and uses the production configuration.
func NewRoleAuthZConfig() *RoleAuthZConfig {
	return &RoleAuthZConfig{
		Enabled:         true,
		RolesConfigFile: "config/admin-api-roles-prod.yaml",
	}
}

// AddFlags adds required flags for the role authZ configuration.
func (c *RoleAuthZConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.RolesConfigFile, "role-authz-config-file", c.RolesConfigFile,
		"Admin API roles configuration file containing list of required role per API method")
	fs.BoolVar(&c.Enabled, "enable-role-authz", c.Enabled, "Enable admin API role authZ")
}

// ReadFiles will read and validate the contents of the configuration file.
func (c *RoleAuthZConfig) ReadFiles() error {
	if c.Enabled {
		if err := readRoleAuthZConfigFile(c.RolesConfigFile, c.RolesConfig); err != nil {
			return err
		}
		return validateRolesConfiguration(c.RolesConfig)
	}
	return nil
}

// GetRoleMapping will create a map of the required roles. The key will be the HTTP method and value will be a list of
// allowed roles for that specific HTTP method.
func (c *RoleAuthZConfig) GetRoleMapping() map[string][]string {
	roleMapping := make(map[string][]string, len(c.RolesConfig))

	for _, config := range c.RolesConfig {
		roleMapping[config.HTTPMethod] = config.RoleNames
	}

	return roleMapping
}

func readRoleAuthZConfigFile(file string, val []RolesConfiguration) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "reading role authz config")
	}

	if err := yaml.UnmarshalStrict([]byte(fileContents), val); err != nil {
		return errors.Wrap(err, "unmarshalling role authz config")
	}

	return nil
}

var allowedHTTPMethods = []string{http.MethodGet, http.MethodPatch, http.MethodDelete}

func validateRolesConfiguration(configs []RolesConfiguration) error {
	for _, config := range configs {
		if !shared.Contains(allowedHTTPMethods, config.HTTPMethod) {
			return fmt.Errorf("invalid http method used %q, expected to be one of [%s]",
				config.HTTPMethod, strings.Join(allowedHTTPMethods, ","))
		}
	}
	return nil
}
