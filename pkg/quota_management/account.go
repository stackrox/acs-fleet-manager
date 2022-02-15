package quota_management

type Account struct {
	Username            string `yaml:"username"`
	MaxAllowedInstances int    `yaml:"max_allowed_instances"`
}

func (account Account) IsInstanceCountWithinLimit(count int) bool {
	return count < account.GetMaxAllowedInstances()
}

func (account Account) GetMaxAllowedInstances() int {
	if account.MaxAllowedInstances <= 0 {
		return MaxAllowedInstances
	}

	return account.MaxAllowedInstances
}

type AccountList []Account

func (allowedAccounts AccountList) GetByUsername(username string) (Account, bool) {
	for _, user := range allowedAccounts {
		if username == user.Username {
			return user, true
		}
	}

	return Account{}, false
}
