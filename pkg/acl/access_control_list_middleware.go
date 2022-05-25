package acl

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

type AccessControlListMiddleware struct {
	accessControlListConfig *AccessControlListConfig
}

func NewAccessControlListMiddleware(accessControlListConfig *AccessControlListConfig) *AccessControlListMiddleware {
	middleware := AccessControlListMiddleware{
		accessControlListConfig: accessControlListConfig,
	}
	return &middleware
}

// Middleware handler to authorize users based on the provided ACL configuration
func (middleware *AccessControlListMiddleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		context := r.Context()
		claims, err := auth.GetClaimsFromContext(context)
		if err != nil {
			shared.HandleError(r, w, errors.NewWithCause(errors.ErrorForbidden, err, ""))
			return
		}

		username, _ := claims.GetUsername()

		if middleware.accessControlListConfig.EnableDenyList {
			userIsDenied := middleware.accessControlListConfig.DenyList.IsUserDenied(username)
			if userIsDenied {
				shared.HandleError(r, w, errors.New(errors.ErrorForbidden, "User '%s' is not authorized to access the service.", username))
				return
			}
		}

		orgId, _ := claims.GetOrgId()

		// If the users claim has an orgId, resources should be filtered by their organisation. Otherwise, filter them by owner.
		context = auth.SetFilterByOrganisationContext(context, orgId != "")
		*r = *r.WithContext(context)

		next.ServeHTTP(w, r)
	})
}
