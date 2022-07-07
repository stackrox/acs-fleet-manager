package auth

import (
	"github.com/spf13/pflag"
)

var (
	// OCM token claim keys.
	tenantUsernameClaim string = "username"
	tenantIdClaim       string = "org_id"
	tenantOrgAdminClaim string = "is_org_admin"

	// sso.redhat.com token claim keys.
	alternateTenantUsernameClaim string = "preferred_username"
	tenantUserIdClaim            string = "account_id"
	tenantSubClaim               string = "sub"
	// Only service accounts that have been created via the service_accounts API have this claim set.
	alternateTenantIdClaim string = "rh-org-id"
)

type ContextConfig struct {
}

func NewContextConfig() *ContextConfig {
	return &ContextConfig{}
}

func (c *ContextConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&tenantUsernameClaim, "tenant-username-claim", tenantUsernameClaim,
		"Token claims key to retrieve the corresponding user principal.")
	fs.StringVar(&tenantIdClaim, "tenant-id-claim", tenantIdClaim,
		"Token claims key to retrieve the corresponding organisation ID.")
	fs.StringVar(&alternateTenantIdClaim, "alternate-tenant-id-claim", alternateTenantIdClaim,
		"Token claims key to retrieve the corresponding organisation ID using an alternative claim.")
	fs.StringVar(&tenantOrgAdminClaim, "tenant-org-admin-claim", tenantOrgAdminClaim,
		"Token claims key to retrieve the corresponding organisation admin role.")
	fs.StringVar(&alternateTenantUsernameClaim, "alternate-tenant-username-claim", alternateTenantUsernameClaim,
		"Token claims key to retrieve the corresponding user principal using an alternative claim.")
	fs.StringVar(&tenantUserIdClaim, "tenant-user-id-claim", tenantUserIdClaim,
		"Token claims key to retrieve the corresponding  Account ID.")

}

func (c *ContextConfig) ReadFiles() error {
	return nil
}
