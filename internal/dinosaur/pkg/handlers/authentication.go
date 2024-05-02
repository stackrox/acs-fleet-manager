package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/routes"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type compositeAuthenticationHandler struct {
	defaultHandler    http.Handler
	privateAPIHandler http.Handler
	adminAPIHandler   http.Handler
}

func (h *compositeAuthenticationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, routes.AdminAPIPrefix) {
		h.adminAPIHandler.ServeHTTP(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, routes.PrivateAPIPrefix) {
		h.privateAPIHandler.ServeHTTP(w, r)
		return
	}
	h.defaultHandler.ServeHTTP(w, r)
}

// NewAuthenticationHandler creates a new instance of authentication handler
func NewAuthenticationHandler(IAMConfig *iam.IAMConfig, next http.Handler) (http.Handler, error) {
	authnLogger, err := sdk.NewGlogLoggerBuilder().
		InfoV(glog.Level(1)).
		DebugV(glog.Level(5)).
		Build()

	if err != nil {
		return nil, fmt.Errorf("unable to create authentication logger: %w", err)
	}

	defaultHandlerBuilder := authentication.NewHandler().
		Logger(authnLogger).
		KeysURL(IAMConfig.JwksURL).                        // ocm JWK JSON web token signing certificates URL
		KeysFile(IAMConfig.JwksFile).                      // ocm JWK backup JSON web token signing certificates
		KeysURL(IAMConfig.RedhatSSORealm.JwksEndpointURI). // sso JWK Cert URL
		Error(fmt.Sprint(errors.ErrorUnauthenticated)).
		Service(errors.ErrorCodePrefix).
		Public(fmt.Sprintf("^%s/%s/?$", routes.APIEndpoint, routes.FleetManagementAPIPrefix)).
		Public(fmt.Sprintf("^%s/%s/%s/?$", routes.APIEndpoint, routes.FleetManagementAPIPrefix, routes.Version)).
		Public(fmt.Sprintf("^%s/%s/%s/openapi/?$", routes.APIEndpoint, routes.FleetManagementAPIPrefix, routes.Version)).
		Public(fmt.Sprintf("^%s/%s/%s/errors/?[0-9]*", routes.APIEndpoint, routes.FleetManagementAPIPrefix, routes.Version))

	// Add additional JWKS endpoints to the builder if there are any.
	for _, jwksEndpointURI := range IAMConfig.AdditionalSSOIssuers.JWKSURIs {
		defaultHandlerBuilder.KeysURL(jwksEndpointURI)
	}

	defaultHandler, err := defaultHandlerBuilder.Next(next).Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create default authN handler: %w", err)
	}

	privateAPIHandlerBuilder := authentication.NewHandler().
		Logger(authnLogger).
		KeysURL(IAMConfig.RedhatSSORealm.JwksEndpointURI).
		Error(fmt.Sprint(errors.ErrorUnauthenticated)).
		Service(errors.ErrorCodePrefix)

	// Add additional JWKS endpoints to the builder if there are any.
	for _, jwksEndpointURI := range IAMConfig.DataPlaneOIDCIssuers.JWKSURIs {
		privateAPIHandlerBuilder.KeysURL(jwksEndpointURI)
	}

	privateAPIHandler, err := privateAPIHandlerBuilder.Next(next).Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create private authN handler: %w", err)
	}

	adminAPIHandler, err := authentication.NewHandler().
		Logger(authnLogger).
		KeysURL(IAMConfig.InternalSSORealm.JwksEndpointURI). // internal sso (auth.redhat.com) JWK Cert URL
		Error(fmt.Sprint(errors.ErrorUnauthenticated)).
		Service(errors.ErrorCodePrefix).
		Next(next).
		Build()

	if err != nil {
		return nil, fmt.Errorf("unable to create admin authN handler: %w", err)
	}

	return &compositeAuthenticationHandler{
		defaultHandler:    defaultHandler,
		privateAPIHandler: privateAPIHandler,
		adminAPIHandler:   adminAPIHandler,
	}, nil
}
