package fleetmanager

import (
	"fmt"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"

	"github.com/pkg/errors"
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
	client redhatsso.SSOClient
}

type rhSSOAuthFactory struct{}

// GetName gets the name of the factory.
func (f *rhSSOAuthFactory) GetName() string {
	return rhSSOAuthName
}

// CreateAuth ...
func (f *rhSSOAuthFactory) CreateAuth(o Option) (Auth, error) {
	client := redhatsso.NewSSOClient(&iam.IAMConfig{}, &iam.IAMRealmConfig{
		BaseURL:          o.Sso.Endpoint,
		Realm:            o.Sso.Realm,
		ClientID:         o.Sso.ClientID,
		ClientSecret:     o.Sso.ClientSecret, //pragma: allowlist secret
		TokenEndpointURI: fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", o.Sso.Endpoint, o.Sso.Realm),
		JwksEndpointURI:  fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/certs", o.Sso.Endpoint, o.Sso.Realm),
		APIEndpointURI:   fmt.Sprintf("/auth/realms/%s", o.Sso.Realm),
	})
	return &rhSSOAuth{
		client: client,
	}, nil
}

// AddAuth add auth token to the request retrieved from Red Hat SSO.
func (r *rhSSOAuth) AddAuth(req *http.Request) error {
	token, err := r.client.GetToken()
	if err != nil {
		return errors.Wrap(err, "getting token from RH SSO")
	}
	setBearer(req, token)
	return nil
}
