package auth

import (
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// UseFleetShardAuthorizationMiddleware ...
func UseFleetShardAuthorizationMiddleware(router *mux.Router, iamConfig *iam.IAMConfig,
	fleetShardAuthZConfig *FleetShardAuthZConfig) {
	router.Use(
		NewRequireIssuerMiddleware().RequireIssuer(iamConfig.GetDataPlaneIssuerURIs(), errors.ErrorNotFound),
		CheckAudience(fleetShardAuthZConfig.AllowedAudiences),
		checkSubject(fleetShardAuthZConfig.AllowedSubjects),
	)
}

func checkSubject(allowedSubjects []string) mux.MiddlewareFunc {
	return checkClaim(tenantSubClaim, (*ACSClaims).GetSubject, allowedSubjects)
}

// CheckAudience is a middleware to check if the aud claim in a given request
// matches the allowedAudiences
func CheckAudience(allowedAudiences []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				// Deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are
				// not authorised. Otherwise, we would leak information about existing cluster IDs, since the path
				// of the request is /agent-clusters/<id>.
				shared.HandleError(request, writer, errors.NotFound(""))
				return
			}

			for _, audience := range allowedAudiences {
				if claims.VerifyAudience(audience) {
					next.ServeHTTP(writer, request)
					return
				}
			}

			audience, _ := claims.GetAudience()
			glog.Infof("none of the audiences [%s] in the access token is not in the list of allowed values [%s]",
				strings.Join(audience, ","), strings.Join(allowedAudiences, ","))

			shared.HandleError(request, writer, errors.NotFound(""))
		})
	}
}

type claimsGetter func(*ACSClaims) (string, error)

func checkClaim(claimName string, getter claimsGetter, allowedValues ClaimValues) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				// Deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are
				// not authorised. Otherwise, we would leak information about existing cluster IDs, since the path
				// of the request is /agent-clusters/<id>.
				shared.HandleError(request, writer, errors.NotFound(""))
				return
			}

			claimValue, _ := getter(&claims)
			if allowedValues.Contains(claimValue) {
				next.ServeHTTP(writer, request)
				return
			}

			glog.Infof("%s %q is not in the list of allowed values [%s]",
				claimName, claimValue, strings.Join(allowedValues, ","))

			shared.HandleError(request, writer, errors.NotFound(""))
		})
	}
}
