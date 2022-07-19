package quota_management

// Account ...
type Account struct {
	Username            string `yaml:"username"`
	MaxAllowedInstances int    `yaml:"max_allowed_instances"`
}

// IsInstanceCountWithinLimit ...
func (account Account) IsInstanceCountWithinLimit(count int) bool {
	return count < account.GetMaxAllowedInstances()
}

// GetMaxAllowedInstances ...
func (account Account) GetMaxAllowedInstances() int {
	if account.MaxAllowedInstances <= 0 {
		return MaxAllowedInstances
	}

	return account.MaxAllowedInstances
}

// AccountList ...
type AccountList []Account

// GetByUsername ...
func (allowedAccounts AccountList) GetByUsername(username string) (Account, bool) {
	for _, user := range allowedAccounts {
		if username == user.Username {
			return user, true
		}
	}

	return Account{}, false
}
