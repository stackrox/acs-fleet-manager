package quota_management

// Organisation ...
type Organisation struct {
	Id                  string      `yaml:"id"`
	AnyUser             bool        `yaml:"any_user"`
	MaxAllowedInstances int         `yaml:"max_allowed_instances"`
	RegisteredUsers     AccountList `yaml:"registered_users"`
}

// IsUserRegistered ...
func (org Organisation) IsUserRegistered(username string) bool {
	if !org.HasUsersRegistered() {
		return org.AnyUser
	}
	_, found := org.RegisteredUsers.GetByUsername(username)
	return found
}

// HasUsersRegistered ...
func (org Organisation) HasUsersRegistered() bool {
	return len(org.RegisteredUsers) > 0
}

// IsInstanceCountWithinLimit ...
func (org Organisation) IsInstanceCountWithinLimit(count int) bool {
	return count < org.GetMaxAllowedInstances()
}

// GetMaxAllowedInstances ...
func (org Organisation) GetMaxAllowedInstances() int {
	if org.MaxAllowedInstances <= 0 {
		return MaxAllowedInstances
	}

	return org.MaxAllowedInstances
}

// OrganisationList ...
type OrganisationList []Organisation

// GetById ...
func (orgList OrganisationList) GetById(Id string) (Organisation, bool) {
	for _, organisation := range orgList {
		if Id == organisation.Id {
			return organisation, true
		}
	}

	return Organisation{}, false
}
