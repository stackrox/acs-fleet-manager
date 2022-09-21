package fleetmanager

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/stackrox/rox/pkg/utils"

	"github.com/pkg/errors"
)

// Auth will handle adding authentication information to HTTP requests.
type Auth interface {
	// AddAuth will add authentication information to the request, i.e. in the form of the Authorization header.
	AddAuth(req *http.Request) error
}

type authFactory interface {
	GetName() string
	CreateAuth(o option) (Auth, error)
}

var authFactoryRegistry map[string]authFactory

func init() {
	authFactoryRegistry = map[string]authFactory{
		ocmFactory.GetName():         ocmFactory,
		rhSSOFactory.GetName():       rhSSOFactory,
		staticTokenFactory.GetName(): staticTokenFactory,
	}
}

// NewAuth will return Auth that can be used to add authentication of a specific AuthType to be added to HTTP requests.
func NewAuth(t string, opts ...AuthOption) (Auth, error) {
	authOption := &option{}
	for _, opt := range opts {
		opt(authOption)
	}

	factory, exists := authFactoryRegistry[t]
	if !exists {
		return nil, errors.Errorf("invalid auth type found: %q, must be one of [%s]",
			t, strings.Join(getAllAuthTypes(), ","))
	}
	auth, err := factory.CreateAuth(*authOption)
	if err != nil {
		return auth, fmt.Errorf("creating Auth: %w", err)
	}
	return auth, nil
}

// option for the different Auth types.
type option struct {
	Sso    RhSsoOption
	Ocm    OCMOption
	Static StaticOption
}

// RhSsoOption for the RH SSO Auth type.
type RhSsoOption struct {
	TokenFile string `env:"RHSSO_TOKEN_FILE" envDefault:"/run/secrets/rhsso-token/token"`
}

// OCMOption for the OCM Auth type.
type OCMOption struct {
	RefreshToken string `env:"OCM_TOKEN"`
}

// StaticOption for the Static Auth type.
type StaticOption struct {
	StaticToken string `env:"STATIC_TOKEN"`
}

// AuthOption to configure the different Auth types.
type AuthOption func(*option)

// WithRhSSOOption will set the options for OCM auth.
func WithRhSSOOption(sso RhSsoOption) AuthOption {
	return func(o *option) {
		if sso.TokenFile != "" {
			o.Sso.TokenFile = sso.TokenFile
		}
	}
}

// WithOCMOption will set the options for OCM auth.
func WithOCMOption(ocm OCMOption) AuthOption {
	return func(o *option) {
		if ocm.RefreshToken != "" {
			o.Ocm.RefreshToken = ocm.RefreshToken
		}
	}
}

// WithStaticOption will set the options for static auth.
func WithStaticOption(static StaticOption) AuthOption {
	return func(o *option) {
		if static.StaticToken != "" {
			o.Static.StaticToken = static.StaticToken
		}
	}
}

// WithOptionFromEnv will override the option values using environment variables.
// Currently, the following are supported:
//   - OCM_TOKEN for the OCM refresh token.
//   - STATIC_TOKEN for the static token.
//   - RHSSO_TOKEN_FILE for the path to the file containing the RH SSO access token.
func WithOptionFromEnv() AuthOption {
	return func(o *option) {
		optFromEnv := &option{}
		utils.Must(env.Parse(optFromEnv))
		overrideValues(o, optFromEnv)
	}
}

// setBearer is a helper to set a bearer token as authorization header on the http.Request.
func setBearer(req *http.Request, token string) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
}

// getAllAuthTypes is a helper used within logging to list the possible values for auth types.
func getAllAuthTypes() []string {
	authTypes := make([]string, 0, len(authFactoryRegistry))
	for authType := range authFactoryRegistry {
		authTypes = append(authTypes, authType)
	}
	return authTypes
}

func overrideValues(orig *option, updated *option) {
	if updated.Sso.TokenFile != "" {
		orig.Sso.TokenFile = updated.Sso.TokenFile
	}
	if updated.Ocm.RefreshToken != "" {
		orig.Ocm.RefreshToken = updated.Ocm.RefreshToken
	}
	if updated.Static.StaticToken != "" {
		orig.Static.StaticToken = updated.Static.StaticToken
	}
}
