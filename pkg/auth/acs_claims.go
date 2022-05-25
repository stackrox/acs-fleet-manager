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
	return "", fmt.Errorf("can't find neither '%s' or '%s' attribute in claims", tenantUsernameClaim, alternateTenantUsernameClaim)
}

func (c *ACSClaims) GetAccountId() (string, error) {
	if (*c)[tenantUserIdClaim] != nil {
		return (*c)[tenantUserIdClaim].(string), nil
	}
	return "", fmt.Errorf("can't find '%s' attribute in claims", tenantUserIdClaim)
}

func (c *ACSClaims) GetOrgId() (string, error) {
	if (*c)[tenantIdClaim] != nil {
		if orgId, ok := (*c)[tenantIdClaim].(string); ok {
			return orgId, nil
		}
	}

	// NOTE: This should be removed once we migrate to sso.redhat.com as it will no longer be needed (TODO: to be removed as part of MGDSTRM-6159)
	if (*c)[alternateTenantIdClaim] != nil {
		if orgId, ok := (*c)[alternateTenantIdClaim].(string); ok {
			return orgId, nil
		}
	}

	return "", fmt.Errorf("can't find neither '%s' or '%s' attribute in claims", tenantIdClaim, alternateTenantIdClaim)
}

func (c *ACSClaims) IsOrgAdmin() bool {
	if (*c)[tenantOrgAdminClaim] != nil {
		return (*c)[tenantOrgAdminClaim].(bool)
	}
	return false
}
