package auth

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

func UseOperatorAuthorisationMiddleware(router *mux.Router, jwkValidIssuerURI string, clusterIdVar string) {
	var requiredRole = "fleetshard_operator"

	router.Use(
		NewRolesAuhzMiddleware().RequireRealmRole(requiredRole, errors.ErrorNotFound),
		checkClusterId(clusterIdVar),
		NewRequireIssuerMiddleware().RequireIssuer([]string{jwkValidIssuerURI}, errors.ErrorNotFound),
	)
}

func checkClusterId(clusterIdVar string) mux.MiddlewareFunc {
	var clusterIdClaimKey = "fleetshard-operator-cluster-id"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			clusterId := mux.Vars(request)[clusterIdVar]
			claims, err := GetClaimsFromContext(ctx)
			if err != nil {
				// deliberately return 404 here so that it will appear as the endpoint doesn't exist if requests are not authorised
				shared.HandleError(request, writer, errors.BadRequest(""))
				return
			}
			if clusterIdInClaim, ok := claims[clusterIdClaimKey].(string); ok {
				if clusterIdInClaim == clusterId {
					next.ServeHTTP(writer, request)
					return
				}
			}
			shared.HandleError(request, writer, errors.DuplicateDinosaurClusterName())
		})
	}
}
