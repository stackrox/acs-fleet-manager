// Package fleetmanager ...
package fleetmanager

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/caarlos0/env/v6"
	"github.com/stackrox/rox/pkg/utils"

	"github.com/pkg/errors"
)

// Auth will handle adding authentication information to HTTP requests.
type Auth interface {
	// AddAuth will add authentication information to the request, e.g. in the form of the Authorization header.
	AddAuth(req *http.Request) error

	// RetrieveIDToken will return the ID token, if available.
	// If the ID token cannot be retrieved, an error will be returned.
	RetrieveIDToken() (string, error)
}

type authFactory interface {
	GetName() string
	CreateAuth(o Option) (Auth, error)
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
func NewAuth(t string, opt Option) (Auth, error) {
	return newAuth(t, opt)
}

func newAuth(t string, opt Option) (Auth, error) {
	factory, exists := authFactoryRegistry[t]
	if !exists {
		return nil, errors.Errorf("invalid auth type found: %q, must be one of [%s]",
			t, strings.Join(getAllAuthTypes(), ","))
	}

	auth, err := factory.CreateAuth(opt)
	if err != nil {
		return auth, fmt.Errorf("creating Auth: %w", err)
	}
	return auth, nil
}

// NewRHSSOAuth will return Auth that uses RH SSO to provide authentication for HTTP requests.
func NewRHSSOAuth(opt RHSSOOption) (Auth, error) {
	return newAuth(rhSSOFactory.GetName(), Option{Sso: opt})
}

// NewOCMAuth will return Auth that uses OCM to provide authentication for HTTP requests.
func NewOCMAuth(opt OCMOption) (Auth, error) {
	return newAuth(ocmFactory.GetName(), Option{Ocm: opt})
}

// NewStaticAuth will return Auth that uses a static token to provide authentication for HTTP requests.
func NewStaticAuth(opt StaticOption) (Auth, error) {
	return newAuth(staticTokenFactory.GetName(), Option{Static: opt})
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
