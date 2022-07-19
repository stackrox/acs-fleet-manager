package account

import (
	sdk "github.com/openshift-online/ocm-sdk-go"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

type accountService struct {
	connection *sdk.Connection
}

var _ AccountService = &accountService{}

// NewAccountService ...
func NewAccountService(connection *sdk.Connection) AccountService {
	return &accountService{
		connection,
	}
}

// SearchOrganizations ...
func (as *accountService) SearchOrganizations(filter string) (*OrganizationList, error) {
	items, err := as.searchOrganizations(filter)

	if err != nil {
		return nil, err
	}

	return convertOrganizationList(items), nil
}

func (as *accountService) searchOrganizations(filter string) (*v1.OrganizationList, error) {
	res, err := as.connection.AccountsMgmt().V1().Organizations().List().Search(filter).Send()
	if err != nil {
		return nil, err
	}

	return res.Items(), nil
}

// GetOrganization ...
func (as *accountService) GetOrganization(filter string) (*Organization, error) {
	organizationList, err := as.searchOrganizations(filter)
	if err != nil {
		return nil, err
	}

	if organizationList.Len() == 0 {
		return nil, nil
	}
	return convertOrganization(organizationList.Get(0)), nil
}

func convertOrganization(o *v1.Organization) *Organization {
	return &Organization{
		ID:            o.ID(),
		Name:          o.Name(),
		AccountNumber: o.EbsAccountID(),
		ExternalID:    o.ExternalID(),
		CreatedAt:     o.CreatedAt(),
		UpdatedAt:     o.UpdatedAt(),
	}
}

func convertOrganizationList(organizationList *v1.OrganizationList) *OrganizationList {
	var list []*Organization
	organizationList.Each(func(item *v1.Organization) bool {
		list = append(list, convertOrganization(item))
		return true
	})

	return &OrganizationList{
		items: list,
	}
}
