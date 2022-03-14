package quota_management

import (
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"os"
)

type QuotaManagementListConfig struct {
	QuotaList                  RegisteredUsersListConfiguration
	QuotaListConfigFile        string
	EnableInstanceLimitControl bool
}

func NewQuotaManagementListConfig() *QuotaManagementListConfig {
	return &QuotaManagementListConfig{
		QuotaListConfigFile:        "config/quota-management-list-configuration.yaml",
		EnableInstanceLimitControl: false,
	}
}

func (c *QuotaManagementListConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.QuotaListConfigFile, "quota-management-list-config-file", c.QuotaListConfigFile, "QuotaList configuration file")
	fs.IntVar(&MaxAllowedInstances, "max-allowed-instances", MaxAllowedInstances, "Default maximum number of allowed instances that can be created by a user")
	fs.BoolVar(&c.EnableInstanceLimitControl, "enable-instance-limit-control", c.EnableInstanceLimitControl, "Enable to enforce limits on how much instances a user can create")
}

func (c *QuotaManagementListConfig) ReadFiles() error {
	// TODO: we should avoid reading the file if quota-type is not quota-management-list
	// ATM, since the quota-type is inside DinosaurConfig and DinosaurConfig is not accessible from here, I will leave this for a
	// future implementation
	err := readQuotaManagementListConfigFile(c.QuotaListConfigFile, &c.QuotaList)

	if os.IsNotExist(err) {
		logger.Logger.Warningf("Configuration file for quota-management-list not found: '%s'", c.QuotaListConfigFile)
		return nil
	}

	return err
}

func (c *QuotaManagementListConfig) GetAllowedAccountByUsernameAndOrgId(username string, orgId string) (Account, bool) {
	var user Account
	var found bool
	org, _ := c.QuotaList.Organisations.GetById(orgId)
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
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
