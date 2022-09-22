package fleetmanager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	rhSSOAuthName = "RHSSO"
)

var (
	_            authFactory = (*rhSSOAuthFactory)(nil)
	_            Auth        = (*rhSSOAuth)(nil)
	rhSSOFactory             = &rhSSOAuthFactory{}
)

type rhSSOAuth struct {
	tokenSource oauth2.TokenSource
}

type rhSSOAuthFactory struct{}

// GetName gets the name of the factory.
func (f *rhSSOAuthFactory) GetName() string {
	return rhSSOAuthName
}

// CreateAuth creates an Auth using RH SSO.
func (f *rhSSOAuthFactory) CreateAuth(o Option) (Auth, error) {
	cfg := clientcredentials.Config{
		ClientID:     o.Sso.ClientID,
		ClientSecret: o.Sso.ClientSecret, //pragma: allowlist secret
		TokenURL:     fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", o.Sso.Endpoint, o.Sso.Realm),
		Scopes:       []string{"openid"},
	}
	return &rhSSOAuth{
		tokenSource: cfg.TokenSource(context.Background()),
	}, nil
}

// AddAuth add auth token to the request retrieved from Red Hat SSO.
func (r *rhSSOAuth) AddAuth(req *http.Request) error {
	token, err := r.tokenSource.Token()
	if err != nil {
		return errors.Wrap(err, "retrieving token from token source")
	}
	setBearer(req, token.AccessToken)
	return nil
}
