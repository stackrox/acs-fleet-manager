// Package auth ...
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

// VerifyAudience wraps jwt.VerifyAudience
func (c *ACSClaims) VerifyAudience(cmp string, req bool) bool {
	return jwt.MapClaims(*c).VerifyAudience(cmp, req)
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
	if accountID, ok := (*c)[tenantAccountIDClaim].(string); ok {
		return accountID, nil
	}
	return "", fmt.Errorf("can't find %q attribute in claims", tenantAccountIDClaim)
}

// GetUserID returns the user id of the Red Hat account associated to the token.
func (c *ACSClaims) GetUserID() (string, error) {
	if idx, val := arrays.FindFirst(func(x interface{}) bool { return x != nil },
		(*c)[tenantUserIDClaim], (*c)[alternateTenantUserIDClaim]); idx != -1 {
		if userID, ok := val.(string); ok {
			return userID, nil
		}
	}

	return "", fmt.Errorf("can't find neither %q or %q attribute in claims",
		tenantUserIDClaim, alternateTenantUserIDClaim)
}

// GetAlternateUserID ...
func (c *ACSClaims) GetAlternateUserID() (string, error) {
	if alternateSub, ok := (*c)[alternateSubClaim].(string); ok {
		return alternateSub, nil
	}
	return "", fmt.Errorf("can't find %q attribute in claims", alternateSubClaim)
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

// GetSubject returns the subject claim of the token. It identifies the principal authenticated by the token.
func (c *ACSClaims) GetSubject() (string, error) {
	if sub, ok := (*c)[tenantSubClaim].(string); ok {
		return sub, nil
	}

	return "", fmt.Errorf("can't find %q attribute in claims", tenantSubClaim)
}

// GetAudience returns the audience claim of the token. It identifies the token consumer.
func (c *ACSClaims) GetAudience() ([]string, error) {
	aud := make([]string, 0)
	switch v := (*c)[audienceClaim].(type) {
	case string:
		aud = append(aud, v)
	case []string:
		aud = v
	case []interface{}:
		for _, a := range v {
			vs, ok := a.(string)
			if !ok {
				return nil, fmt.Errorf("can't parse part of the audience claim: %q", a)
			}
			aud = append(aud, vs)
		}
	default:
		return nil, fmt.Errorf("can't parse the audience claim: %q", v)
	}
	return aud, nil
}

// IsOrgAdmin ...
func (c *ACSClaims) IsOrgAdmin() bool {
	isOrgAdmin, _ := (*c)[tenantOrgAdminClaim].(bool)
	return isOrgAdmin
}
