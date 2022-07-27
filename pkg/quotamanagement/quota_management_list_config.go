package quotamanagement

import (
	"fmt"
	"io/fs"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"gopkg.in/yaml.v2"
)

// QuotaManagementListConfig ...
type QuotaManagementListConfig struct {
	QuotaList                  RegisteredUsersListConfiguration
	QuotaListConfigFile        string
	EnableInstanceLimitControl bool
}

// NewQuotaManagementListConfig ...
func NewQuotaManagementListConfig() *QuotaManagementListConfig {
	return &QuotaManagementListConfig{
		QuotaListConfigFile:        "config/quota-management-list-configuration.yaml",
		EnableInstanceLimitControl: false,
	}
}

// AddFlags ...
func (c *QuotaManagementListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.QuotaListConfigFile, "quota-management-list-config-file", c.QuotaListConfigFile, "QuotaList configuration file")
	fs.IntVar(&MaxAllowedInstances, "max-allowed-instances", MaxAllowedInstances, "Default maximum number of allowed instances that can be created by a user")
	fs.BoolVar(&c.EnableInstanceLimitControl, "enable-instance-limit-control", c.EnableInstanceLimitControl, "Enable to enforce limits on how much instances a user can create")
}

// ReadFiles ...
func (c *QuotaManagementListConfig) ReadFiles() error {
	// TODO: we should avoid reading the file if quota-type is not quota-management-list
	// ATM, since the quota-type is inside DinosaurConfig and DinosaurConfig is not accessible from here, I will leave this for a
	// future implementation
	err := readQuotaManagementListConfigFile(c.QuotaListConfigFile, &c.QuotaList)

	if errors.Is(err, fs.ErrNotExist) {
		logger.Logger.Warningf("Configuration file for quota-management-list not found: '%s'", c.QuotaListConfigFile)
		return nil
	}

	return err
}

// GetAllowedAccountByUsernameAndOrgID ...
func (c *QuotaManagementListConfig) GetAllowedAccountByUsernameAndOrgID(username string, orgID string) (Account, bool) {
	var user Account
	var found bool
	org, _ := c.QuotaList.Organisations.GetByID(orgID)
	user, found = org.RegisteredUsers.GetByUsername(username)
	if found {
		return user, found
	}
	return c.QuotaList.ServiceAccounts.GetByUsername(username)
}

// Read the contents of file into the quota list config
func readQuotaManagementListConfigFile(file string, val *RegisteredUsersListConfiguration) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading quota management list config file: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), val)
	if err != nil {
		return fmt.Errorf("unmarshalling quota management list config file: %w", err)
	}
	return nil
}
