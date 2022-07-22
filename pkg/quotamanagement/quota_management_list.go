package quotamanagement

// MaxAllowedInstances ...
var MaxAllowedInstances = 1

// GetDefaultMaxAllowedInstances - Returns the default max allowed instances for both internal (users and orgs in quota list config) and external users
func GetDefaultMaxAllowedInstances() int {
	return MaxAllowedInstances
}

// QuotaManagementListItem ...
type QuotaManagementListItem interface {
	// IsInstanceCountWithinLimit returns true if the given count is within limits
	IsInstanceCountWithinLimit(count int) bool
	// GetMaxAllowedInstances returns maximum number of allowed instances.
	GetMaxAllowedInstances() int
}

// RegisteredUsersListConfiguration ...
type RegisteredUsersListConfiguration struct {
	Organisations   OrganisationList `yaml:"registered_users_per_organisation"`
	ServiceAccounts AccountList      `yaml:"registered_service_accounts"`
}
