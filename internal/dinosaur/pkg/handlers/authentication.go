package handlers

import (
	"fmt"

	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/routes"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
)

func NewAuthenticationBuilder(ServerConfig *server.ServerConfig, IAMConfig *iam.IAMConfig) (*authentication.HandlerBuilder, error) {

	authnLogger, err := sdk.NewGlogLoggerBuilder().
		InfoV(glog.Level(1)).
		DebugV(glog.Level(5)).
		Build()

	if err != nil {
		return nil, pkgErrors.Wrap(err, "unable to create authentication logger")
	}

	authenticationBuilder := authentication.NewHandler()

	// Add additional JWKS endpoints to the builder if there are any.
	for _, jwksEndpointURI := range IAMConfig.AdditionalSSOIssuers.JWKSURIs {
		authenticationBuilder.KeysURL(jwksEndpointURI)
	}

	return authenticationBuilder.
			Logger(authnLogger).
			KeysURL(ServerConfig.JwksURL).                     // ocm JWK JSON web token signing certificates URL
			KeysFile(ServerConfig.JwksFile).                   // ocm JWK backup JSON web token signing certificates
			KeysURL(IAMConfig.RedhatSSORealm.JwksEndpointURI). // sso JWK Cert URL
			Error(fmt.Sprint(errors.ErrorUnauthenticated)).
			Service(errors.ERROR_CODE_PREFIX).
			Public(fmt.Sprintf("^%s/%s/?$", routes.ApiEndpoint, routes.DinosaursFleetManagementApiPrefix)).
			Public(fmt.Sprintf("^%s/%s/%s/?$", routes.ApiEndpoint, routes.DinosaursFleetManagementApiPrefix, routes.Version)).
			Public(fmt.Sprintf("^%s/%s/%s/openapi/?$", routes.ApiEndpoint, routes.DinosaursFleetManagementApiPrefix, routes.Version)).
			Public(fmt.Sprintf("^%s/%s/%s/errors/?[0-9]*", routes.ApiEndpoint, routes.DinosaursFleetManagementApiPrefix, routes.Version)),
		nil
}
