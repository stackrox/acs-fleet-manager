package auth

import (
	"net/http"
	"strings"

	"github.com/golang/glog"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/golang-jwt/jwt/v4"
)

const (
	FleetManagerAdminReadRole  = "fleet-manager-admin-read"
	FleetManagerAdminWriteRole = "fleet-manager-admin-write"
	FleetManagerAdminFullRole  = "fleet-manager-admin-full"
)

// RolesAuthorizationMiddleware can be used to perform RBAC authorization checks on endpoints
type RolesAuthorizationMiddleware interface {
	// RequireRealmRole will check the given realm role exists in the request token
	RequireRealmRole(roleName string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
	// RequireRolesForMethods will check that at least one of the realm roles exists in the request token based on the http method in the request
	RequireRolesForMethods(roles map[string][]string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler
}

type rolesAuthMiddleware struct{}

var _ RolesAuthorizationMiddleware = &rolesAuthMiddleware{}

func NewRolesAuhzMiddleware() RolesAuthorizationMiddleware {
	return &rolesAuthMiddleware{}
}

func (m *rolesAuthMiddleware) RequireRealmRole(roleName string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			serviceErr := errors.New(code, "")
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}
			roles := getRealmRolesClaim(claims)
			if hasRole(roles, roleName) {
				next.ServeHTTP(writer, request)
			} else {
				shared.HandleError(request, writer, serviceErr)
				return
			}
		})
	}
}

func (m *rolesAuthMiddleware) RequireRolesForMethods(roles map[string][]string, code errors.ServiceErrorCode) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			serviceErr := errors.New(code, "")
			method := request.Method
			allowedRoles, ok := roles[method]
			if !ok {
				// no allowed roles defined for the given method, deny the request by default to be safer
				glog.Infof("no allowed roles defined for method %s, deny the request for url %s", method, request.URL)
				shared.HandleError(request, writer, serviceErr)
				return
			}

			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				shared.HandleError(request, writer, serviceErr)
				return
			}
			realmRoles := getRealmRolesClaim(claims)
			// if the request claim has any realm role that is defined in the `roles` map, the request will be allowed
			for _, r := range allowedRoles {
				if hasRole(realmRoles, r) {
					ctx = SetIsAdminContext(ctx, true)
					request = request.WithContext(ctx)
					next.ServeHTTP(writer, request)
					return
				}
			}
			// no matching roles found, deny the request
			shared.HandleError(request, writer, serviceErr)
		})
	}
}

func getRealmRolesClaim(claims jwt.MapClaims) []string {
	if realmRoles, ok := claims["realm_access"]; ok {
		if roles, ok := realmRoles.(map[string]interface{}); ok {
			if arr, ok := roles["roles"].([]interface{}); ok {
				var r []string
				for _, i := range arr {
					r = append(r, i.(string))
				}
				return r
			}
		}
	}
	return []string{}
}

func hasRole(roles []string, roleName string) bool {
	for _, role := range roles {
		if strings.EqualFold(role, roleName) {
			return true
		}
	}
	return false
}
