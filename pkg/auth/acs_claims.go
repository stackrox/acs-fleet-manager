// Package auth contains the authentication logic for the Fleet Manager API.
package auth

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

const rhacsNamespacePrefix = "rhacs-"

// ACSClaims claims of the JWT access token specific to ACS.
type ACSClaims jwt.MapClaims

// VerifyIssuer verifies the issuer claim of the access token
func (c *ACSClaims) VerifyIssuer(cmp string, reqired bool) bool {
	return jwt.MapClaims(*c).VerifyIssuer(cmp, reqired)
}

// VerifyAudience verifies the audience claim of the access token.
func (c *ACSClaims) VerifyAudience(cmp string) bool {
	return jwt.MapClaims(*c).VerifyAudience(cmp, true)
}

// GetUsername returns the username claim of the token or error if the claim can't be found.
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

// GetAccountID returns the account ID claim of the access token.
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

// GetAlternateUserID returns the alternate user ID claim of the access token.
func (c *ACSClaims) GetAlternateUserID() (string, error) {
	if alternateSub, ok := (*c)[alternateSubClaim].(string); ok {
		return alternateSub, nil
	}
	return "", fmt.Errorf("can't find %q attribute in claims", alternateSubClaim)
}

// GetOrgID returns organization ID claim of the access token.
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
			if vs, ok := a.(string); !ok {
				userID, _ := c.GetUserID()
				glog.V(5).Infof("can't parse part of the audience claim for user %q: %q", userID, a)
			} else {
				aud = append(aud, vs)
			}
		}
	default:
		return nil, fmt.Errorf("can't parse the audience claim: %q", v)
	}
	return aud, nil
}

// IsOrgAdmin returns true if the access token indicates that the owner of this token is an organization admin.
func (c *ACSClaims) IsOrgAdmin() bool {
	isOrgAdmin, _ := (*c)[tenantOrgAdminClaim].(bool)
	return isOrgAdmin
}

// GetTenantID returns tenantID parsed from subject claim of the token.
// The subject claim can consist of colon separated keys.
// This method assumes that subject has key which starts with `rhacs-` prefix.
func (c *ACSClaims) GetTenantID() (string, error) {
	sub, err := c.GetSubject()
	if err != nil {
		return "", fmt.Errorf("can't find subject: %v", err)
	}
	for _, key := range strings.Split(sub, ":") {
		if strings.HasPrefix(key, rhacsNamespacePrefix) {
			return strings.TrimPrefix(key, rhacsNamespacePrefix), nil
		}
	}
	return "", fmt.Errorf("can't find tenant ID in subject %q", sub)
}

// CheckAllowedOrgIDs is a middleware to check if org id claim in a
// given request matches the allowedOrgIDs
func CheckAllowedOrgIDs(allowedOrgIDs []string) mux.MiddlewareFunc {
	return checkClaim(tenantIDClaim, (*ACSClaims).GetOrgID, allowedOrgIDs)
}
