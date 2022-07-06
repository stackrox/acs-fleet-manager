package auth

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

type ACSClaims jwt.MapClaims

func (c *ACSClaims) VerifyIssuer(cmp string, req bool) bool {
	return jwt.MapClaims(*c).VerifyIssuer(cmp, req)
}

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

func (c *ACSClaims) GetAccountId() (string, error) {
	if (*c)[tenantUserIdClaim] != nil {
		if accountId, ok := (*c)[tenantUserIdClaim].(string); ok {
			return accountId, nil
		}
	}
	return "", fmt.Errorf("can't find %q attribute in claims", tenantUserIdClaim)
}

func (c *ACSClaims) GetOrgId() (string, error) {
	if idx, val := arrays.FindFirst(func(x interface{}) bool { return x != nil },
		(*c)[tenantIdClaim], (*c)[alternateTenantIdClaim]); idx != -1 {
		if orgId, ok := val.(string); ok {
			return orgId, nil
		}
	}

	return "", fmt.Errorf("can't find neither %q or %q attribute in claims",
		tenantIdClaim, alternateTenantIdClaim)
}

func (c *ACSClaims) GetUserId() (string, error) {
	if (*c)[tenantSubClaim] != nil {
		if sub, ok := (*c)[tenantSubClaim].(string); ok {
			return sub, nil
		}
	}

	return "", fmt.Errorf("can't find %q attribute in claims", tenantSubClaim)
}

func (c *ACSClaims) IsOrgAdmin() bool {
	if (*c)[tenantOrgAdminClaim] != nil {
		if isOrgAdmin, ok := (*c)[tenantOrgAdminClaim].(bool); ok {
			return isOrgAdmin
		}
	}
	return false
}
