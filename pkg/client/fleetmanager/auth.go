// Package fleetmanager ...
package fleetmanager

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// Auth will handle adding authentication information to HTTP requests.
type Auth interface {
	// AddAuth will add authentication information to the request, e.g. in the form of the Authorization header.
	AddAuth(req *http.Request) error
}

type authFactory interface {
	GetName() string
	CreateAuth(ctx context.Context, o Option) (Auth, error)
}

// Option for the different Auth types.
type Option struct {
	Sso    RHSSOOption
	Ocm    OCMOption
	Static StaticOption
}

// RHSSOOption for the RH SSO Auth type.
type RHSSOOption struct {
	ClientID     string `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	ClientSecret string `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET"` //pragma: allowlist secret
	Realm        string `env:"RHSSO_REALM" envDefault:"redhat-external"`
	Endpoint     string `env:"RHSSO_ENDPOINT" envDefault:"https://sso.redhat.com"`
}

// OCMOption for the OCM Auth type.
type OCMOption struct {
	RefreshToken string `env:"OCM_TOKEN"`
	EnableLogger bool   `env:"OCM_ENABLE_LOGGER"`
}

// StaticOption for the Static Auth type.
type StaticOption struct {
	StaticToken string `env:"STATIC_TOKEN"`
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
func NewAuth(ctx context.Context, t string, opt Option) (Auth, error) {
	return newAuth(ctx, t, opt)
}

func newAuth(ctx context.Context, t string, opt Option) (Auth, error) {
	factory, exists := authFactoryRegistry[t]
	if !exists {
		return nil, errors.Errorf("invalid auth type found: %q, must be one of [%s]",
			t, strings.Join(getAllAuthTypes(), ","))
	}

	auth, err := factory.CreateAuth(ctx, opt)
	if err != nil {
		return auth, fmt.Errorf("creating Auth: %w", err)
	}
	return auth, nil
}

// NewRHSSOAuth will return Auth that uses RH SSO to provide authentication for HTTP requests.
func NewRHSSOAuth(ctx context.Context, opt RHSSOOption) (Auth, error) {
	return newAuth(ctx, rhSSOFactory.GetName(), Option{Sso: opt})
}

// NewOCMAuth will return Auth that uses OCM to provide authentication for HTTP requests.
func NewOCMAuth(ctx context.Context, opt OCMOption) (Auth, error) {
	return newAuth(ctx, ocmFactory.GetName(), Option{Ocm: opt})
}

// NewStaticAuth will return Auth that uses a static token to provide authentication for HTTP requests.
func NewStaticAuth(ctx context.Context, opt StaticOption) (Auth, error) {
	return newAuth(ctx, staticTokenFactory.GetName(), Option{Static: opt})
}

// OptionFromEnv creates an Option struct with populated values from environment variables.
// See the Option struct tags for the corresponding environment variables supported.
func OptionFromEnv() Option {
	optFromEnv := Option{}
	utils.Must(env.Parse(&optFromEnv))
	return optFromEnv
}

// getAllAuthTypes is a helper used within logging to list the possible values for auth types.
func getAllAuthTypes() []string {
	authTypes := make([]string, 0, len(authFactoryRegistry))
	for authType := range authFactoryRegistry {
		authTypes = append(authTypes, authType)
	}
	sort.Strings(authTypes)
	return authTypes
}
