package auth

import (
	"github.com/golang/glog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

func UseFleetShardAuthorizationMiddleware(router *mux.Router, jwkValidIssuerURI string,
	fleetShardAuthZConfig *FleetShardAuthZConfig) {
	router.Use(
		NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorNotFound),
		checkAllowedOrgIDs(fleetShardAuthZConfig.AllowedOrgIDs),
		NewRequireIssuerMiddleware().RequireIssuer([]string{jwkValidIssuerURI}, errors.ErrorNotFound),
	)
}

func checkAllowedOrgIDs(allowedOrgIDs AllowedOrgIDs) mux.MiddlewareFunc {
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

			orgID, _ := claims.GetOrgId()
			if allowedOrgIDs.IsOrgIDAllowed(orgID) {
				next.ServeHTTP(writer, request)
				return
			}

			glog.Infof("org_id %q is not in the list of allowed org IDs [%s]",
				orgID, strings.Join(allowedOrgIDs, ","))

			shared.HandleError(request, writer, errors.NotFound(""))
		})
	}
}
