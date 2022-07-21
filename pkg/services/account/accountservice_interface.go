package account

// AccountService ...
type AccountService interface {
	SearchOrganizations(filter string) (*OrganizationList, error)
	GetOrganization(filter string) (*Organization, error)
}
