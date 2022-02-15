package auth

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/openshift-online/ocm-sdk-go/authentication"
)

// Context key type defined to avoid collisions in other pkgs using context
// See https://golang.org/pkg/context/#WithValue
type contextKey string

const (
	// Context Keys
	// FilterByOrganisation is used to determine whether resources are filtered by a user's organisation or as an individual owner
	contextFilterByOrganisation contextKey = "filter-by-organisation"
	contextIsAdmin              contextKey = "is_admin"

	// ocm token claim keys
	ocmUsernameKey string = "username"
	ocmOrgIdKey    string = "org_id"
	isOrgAdmin     string = "is_org_admin" // same key used in mas-sso tokens

	// sso.redhat.com token claim keys
	ssoRHUsernameKey  string = "preferred_username" // same key used in mas-sso tokens
	ssoRhAccountIdKey string = "account_id"

	// mas-sso token claim keys
	// NOTE: This should be removed once we migrate to sso.redhat.com as it will no longer be needed (TODO: to be removed as part of MGDSTRM-6159)
	masSsoOrgIdKey = "rh-org-id"
)

func GetUsernameFromClaims(claims jwt.MapClaims) string {
	if claims[ocmUsernameKey] != nil {
		return claims[ocmUsernameKey].(string)
	}

	if claims[ssoRHUsernameKey] != nil {
		return claims[ssoRHUsernameKey].(string)
	}

	return ""
}

func GetAccountIdFromClaims(claims jwt.MapClaims) string {
	if claims[ssoRhAccountIdKey] != nil {
		return claims[ssoRhAccountIdKey].(string)
	}
	return ""
}

func GetOrgIdFromClaims(claims jwt.MapClaims) string {
	if claims[ocmOrgIdKey] != nil {
		if orgId, ok := claims[ocmOrgIdKey].(string); ok {
			return orgId
		}
	}

	// NOTE: This should be removed once we migrate to sso.redhat.com as it will no longer be needed (TODO: to be removed as part of MGDSTRM-6159)
	if claims[masSsoOrgIdKey] != nil {
		if orgId, ok := claims[masSsoOrgIdKey].(string); ok {
			return orgId
		}
	}

	return ""
}

func GetIsOrgAdminFromClaims(claims jwt.MapClaims) bool {
	if claims[isOrgAdmin] != nil {
		return claims[isOrgAdmin].(bool)
	}
	return false
}

func GetIsAdminFromContext(ctx context.Context) bool {
	isAdmin := ctx.Value(contextIsAdmin)
	if isAdmin == nil {
		return false
	}
	return isAdmin.(bool)
}

func SetFilterByOrganisationContext(ctx context.Context, filterByOrganisation bool) context.Context {
	return context.WithValue(ctx, contextFilterByOrganisation, filterByOrganisation)
}

func SetIsAdminContext(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, contextIsAdmin, isAdmin)
}

func GetFilterByOrganisationFromContext(ctx context.Context) bool {
	filterByOrganisation := ctx.Value(contextFilterByOrganisation)
	if filterByOrganisation == nil {
		return false
	}
	return filterByOrganisation.(bool)
}

func SetTokenInContext(ctx context.Context, token *jwt.Token) context.Context {
	return authentication.ContextWithToken(ctx, token)
}

func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, error) {
	var claims jwt.MapClaims
	token, err := authentication.TokenFromContext(ctx)
	if err != nil {
		return claims, fmt.Errorf("failed to get jwt token from context: %v", err)
	}

	if token != nil && token.Claims != nil {
		claims = token.Claims.(jwt.MapClaims)
	}
	return claims, nil
}
