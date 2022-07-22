package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

// ACSClaims ...
type ACSClaims jwt.MapClaims

// VerifyIssuer ...
func (c *ACSClaims) VerifyIssuer(cmp string, req bool) bool {
	return jwt.MapClaims(*c).VerifyIssuer(cmp, req)
}

// GetUsername ...
func (c *ACSClaims) GetUsername() (string, error) {
	if idx, val := arrays.FindFirst(func(x interface{}) bool { return x != nil },
		(*c)[tenantUsernameClaim], (*c)[alternateTenantUsernameClaim]); idx != -1 {
		if userName, ok := val.(string); ok {
			return userName, nil
		}
	}
	return "", fmt.Errorf("can't find neither %q or %q attribute in claims",
		tenantUsernameClaim, alternateTenantUsernameClaim)
}

// GetAccountID ...
func (c *ACSClaims) GetAccountID() (string, error) {
	if accountID, ok := (*c)[tenantUserIDClaim].(string); ok {
		return accountID, nil
	}
	return "", fmt.Errorf("can't find %q attribute in claims", tenantUserIDClaim)
}

// GetOrgID ...
func (c *ACSClaims) GetOrgID() (string, error) {
	if idx, val := arrays.FindFirst(func(x interface{}) bool { return x != nil },
		(*c)[tenantIDClaim], (*c)[alternateTenantIDClaim]); idx != -1 {
		if orgID, ok := val.(string); ok {
			return orgID, nil
		}
	}

	return "", fmt.Errorf("can't find neither %q or %q attribute in claims",
		tenantIDClaim, alternateTenantIDClaim)
}

// GetUserID ...
func (c *ACSClaims) GetUserID() (string, error) {
	if sub, ok := (*c)[tenantSubClaim].(string); ok {
		return sub, nil
	}

	return "", fmt.Errorf("can't find %q attribute in claims", tenantSubClaim)
}

// IsOrgAdmin ...
func (c *ACSClaims) IsOrgAdmin() bool {
	isOrgAdmin, _ := (*c)[tenantOrgAdminClaim].(bool)
	return isOrgAdmin
}
