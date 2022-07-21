package account

import (
	"fmt"
	"time"
)

const (
	mockOrgIDTemplate        = "mock-id-%d"
	mockExternalIDTemplate   = "mock-extid-%d"
	mockEbsAccountIDTemplate = "mock-ebs-%d"
	mockOrgNameTemplate      = "mock-org-%d"
)

// mock returns allowed=true for every request
type mock struct{}

var _ AccountService = &mock{}

// NewMockAccountService ...
func NewMockAccountService() AccountService {
	return &mock{}
}

// SearchOrganizations ...
func (a mock) SearchOrganizations(filter string) (*OrganizationList, error) {
	return buildMockOrganizationList(10), nil
}

// GetOrganization ...
func (a mock) GetOrganization(filter string) (*Organization, error) {
	orgs, _ := a.SearchOrganizations(filter)
	return orgs.Get(0), nil
}

func buildMockOrganizationList(count int) *OrganizationList {
	var mockOrgs []*Organization

	for i := 0; i < count; i++ {
		mockOrgs = append(mockOrgs,
			&Organization{
				ID:            fmt.Sprintf(mockOrgIDTemplate, i),
				Name:          fmt.Sprintf(mockOrgNameTemplate, i),
				AccountNumber: fmt.Sprintf(mockEbsAccountIDTemplate, i),
				ExternalID:    fmt.Sprintf(mockExternalIDTemplate, i),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})
	}

	return &OrganizationList{
		items: mockOrgs,
	}
}
