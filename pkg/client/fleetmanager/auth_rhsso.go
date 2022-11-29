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
	issuer := fmt.Sprintf("%s/auth/realms/%s", o.Sso.Endpoint, o.Sso.Realm)
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving open-id configuration from %q", issuer)
	}

	cfg := clientcredentials.Config{
		ClientID:     o.Sso.ClientID,
		ClientSecret: o.Sso.ClientSecret, //pragma: allowlist secret
		TokenURL:     provider.Endpoint().TokenURL,
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

func (r *rhSSOAuth) RetrieveIDToken() (string, error) {
	t, err := r.tokenSource.Token()
	if err != nil {
		return "", errors.Wrap(err, "retrieving token from token source")
	}

	idTokenRaw := t.Extra("id_token")
	if idTokenRaw == nil {
		return "", errors.New("no ID token could be retrieved")
	}
	if idToken, ok := idTokenRaw.(string); !ok {
		return "", errors.New("ID token was in an unsupported format")
	} else {
		return idToken, nil
	}
}
