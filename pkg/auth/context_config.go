package auth

import (
	"github.com/spf13/pflag"
)

var (
	// OCM token claim keys.
	tenantUsernameClaim = "username"
	tenantIDClaim       = "org_id"
	tenantOrgAdminClaim = "is_org_admin"

	// sso.redhat.com token claim keys.
	alternateTenantUsernameClaim = "preferred_username"
	tenantUserIDClaim            = "account_id"
	tenantSubClaim               = "sub"
	// Only service accounts that have been created via the service_accounts API have this claim set.
	alternateTenantIDClaim = "rh-org-id"
)

// ContextConfig ...
type ContextConfig struct {
}

// NewContextConfig ...
func NewContextConfig() *ContextConfig {
	return &ContextConfig{}
}

// AddFlags ...
func (c *ContextConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&tenantUsernameClaim, "tenant-username-claim", tenantUsernameClaim,
		"Token claims key to retrieve the corresponding user principal.")
	fs.StringVar(&tenantIDClaim, "tenant-id-claim", tenantIDClaim,
		"Token claims key to retrieve the corresponding organisation ID.")
	fs.StringVar(&alternateTenantIDClaim, "alternate-tenant-id-claim", alternateTenantIDClaim,
		"Token claims key to retrieve the corresponding organisation ID using an alternative claim.")
	fs.StringVar(&tenantOrgAdminClaim, "tenant-org-admin-claim", tenantOrgAdminClaim,
		"Token claims key to retrieve the corresponding organisation admin role.")
	fs.StringVar(&alternateTenantUsernameClaim, "alternate-tenant-username-claim", alternateTenantUsernameClaim,
		"Token claims key to retrieve the corresponding user principal using an alternative claim.")
	fs.StringVar(&tenantUserIDClaim, "tenant-user-id-claim", tenantUserIDClaim,
		"Token claims key to retrieve the corresponding  Account ID.")

}

// ReadFiles ...
func (c *ContextConfig) ReadFiles() error {
	return nil
}
