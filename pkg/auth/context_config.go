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
	// This is the EBS account id.
	tenantAccountIDClaim = "account_id"
	// This is the Red Hat user id.
	tenantUserIDClaim = "user_id"
	tenantSubClaim    = "sub"
	// Only service accounts that have been created via the service_accounts API have these claims set.
	// The claims relate to the Red Hat organisation and user that created the service account.
	alternateTenantIDClaim     = "rh-org-id"
	alternateTenantUserIDClaim = "rh-user-id"
)

// ContextConfig ...
type ContextConfig struct{}

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
	fs.StringVar(&tenantAccountIDClaim, "tenant-account-id-claim", tenantAccountIDClaim,
		"Token claims key to retrieve the corresponding EBS account ID.")
	fs.StringVar(&tenantUserIDClaim, "tenant-user-id-claim", tenantUserIDClaim,
		"Token claims key to retrieve the corresponding Red Hat user ID.")
	fs.StringVar(&alternateTenantUserIDClaim, "alternate-tenant-user-id-claim", alternateTenantUserIDClaim,
		"Token claims key to retrieve the corresponding Red Hat user ID using an alternative claim.")
}

// ReadFiles ...
func (c *ContextConfig) ReadFiles() error {
	return nil
}
