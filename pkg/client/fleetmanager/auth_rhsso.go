package fleetmanager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// RHSSOAuthName is the name of the Red Hat Single Sign On authentication method.
	RHSSOAuthName = "RHSSO"
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
	return RHSSOAuthName
}

// CreateAuth creates an Auth using RH SSO.
func (f *rhSSOAuthFactory) CreateAuth(ctx context.Context, o Option) (Auth, error) {
	issuer := fmt.Sprintf("%s/auth/realms/%s", o.Sso.Endpoint, o.Sso.Realm)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving open-id configuration from %q", issuer)
	}

	cfg := clientcredentials.Config{
		ClientID:     o.Sso.ClientID,
		ClientSecret: o.Sso.ClientSecret, //pragma: allowlist secret
		TokenURL:     provider.Endpoint().TokenURL,
		Scopes:       []string{"openid"},
	}
	// This context is used to retrieve tokens at points in time that are arbitrarily
	// far in the future. Current OAuth2 API does not allow bounding the time of an individual
	// token retrieval. https://github.com/golang/oauth2/issues/262
	tokenCtx := context.Background()
	return &rhSSOAuth{
		tokenSource: cfg.TokenSource(tokenCtx),
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
