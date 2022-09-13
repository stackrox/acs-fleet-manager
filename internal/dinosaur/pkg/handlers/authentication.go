package handlers

import (
	"fmt"

	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/routes"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
)

// NewAuthenticationBuilder ...
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
			KeysURL(ServerConfig.JwksURL).                       // ocm JWK JSON web token signing certificates URL
			KeysFile(ServerConfig.JwksFile).                     // ocm JWK backup JSON web token signing certificates
			KeysURL(IAMConfig.RedhatSSORealm.JwksEndpointURI).   // sso JWK Cert URL
			KeysURL(IAMConfig.InternalSSORealm.JwksEndpointURI). // internal sso (auth.redhat.com) JWK Cert URL
			Error(fmt.Sprint(errors.ErrorUnauthenticated)).
			Service(errors.ErrorCodePrefix).
			Public(fmt.Sprintf("^%s/%s/?$", routes.APIEndpoint, routes.DinosaursFleetManagementAPIPrefix)).
			Public(fmt.Sprintf("^%s/%s/%s/?$", routes.APIEndpoint, routes.DinosaursFleetManagementAPIPrefix, routes.Version)).
			Public(fmt.Sprintf("^%s/%s/%s/openapi/?$", routes.APIEndpoint, routes.DinosaursFleetManagementAPIPrefix, routes.Version)).
			Public(fmt.Sprintf("^%s/%s/%s/errors/?[0-9]*", routes.APIEndpoint, routes.DinosaursFleetManagementAPIPrefix, routes.Version)),
		nil
}
