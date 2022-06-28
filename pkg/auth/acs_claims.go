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
	if idx, val := arrays.FindFirst(func(x interface{}) bool { return x != nil }, (*c)[tenantUsernameClaim], (*c)[alternateTenantUsernameClaim]); idx != -1 {
		return val.(string), nil
	}
	return "", fmt.Errorf("can't find neither %q or %q attribute in claims", tenantUsernameClaim, alternateTenantUsernameClaim)
}

func (c *ACSClaims) GetAccountId() (string, error) {
	if (*c)[tenantUserIdClaim] != nil {
		return (*c)[tenantUserIdClaim].(string), nil
	}
	return "", fmt.Errorf("can't find %q attribute in claims", tenantUserIdClaim)
}

func (c *ACSClaims) GetOrgId() (string, error) {
	if (*c)[tenantIdClaim] != nil {
		if orgId, ok := (*c)[tenantIdClaim].(string); ok {
			return orgId, nil
		}
	}

	return "", fmt.Errorf("can't find %q attribute in claims", tenantIdClaim)
}

func (c *ACSClaims) GetSub() (string, error) {
	if (*c)[subClaim] != nil {
		if sub, ok := (*c)[subClaim].(string); ok {
			return sub, nil
		}
	}

	return "", fmt.Errorf("can't find %q attribute in claims", subClaim)
}

func (c *ACSClaims) IsOrgAdmin() bool {
	if (*c)[tenantOrgAdminClaim] != nil {
		return (*c)[tenantOrgAdminClaim].(bool)
	}
	return false
}
