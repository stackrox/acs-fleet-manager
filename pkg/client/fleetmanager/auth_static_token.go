package fleetmanager

import (
	"net/http"

	"github.com/pkg/errors"
)

const (
	// StaticTokenAuthName is the name of the static token authentication method.
	StaticTokenAuthName = "STATIC_TOKEN"
)

var (
	_                  authFactory = (*staticTokenAuthFactory)(nil)
	_                  Auth        = (*staticTokenAuth)(nil)
	staticTokenFactory             = &staticTokenAuthFactory{}
)

type staticTokenAuth struct {
	token string
}

type staticTokenAuthFactory struct{}

// GetName gets the name of the factory.
func (f *staticTokenAuthFactory) GetName() string {
	return StaticTokenAuthName
}

// CreateAuth ...
func (f *staticTokenAuthFactory) CreateAuth(o Option) (Auth, error) {
	staticToken := o.Static.StaticToken
	if staticToken == "" {
		return nil, errors.New("no static token set")
	}
	return &staticTokenAuth{
		token: staticToken,
	}, nil
}

// AddAuth add auth token to the request using a static token.
func (s *staticTokenAuth) AddAuth(req *http.Request) error {
	setBearer(req, s.token)
	return nil
}

func (s *staticTokenAuth) RetrieveIDToken() (string, error) {
	// Since we plan to minimize the usage of the static token auth type, potentially removing it going forward,
	// we skip implementing the `id_token` interface for it.
	return "", errors.New("retrieving ID tokens using the static token auth type is currently not supported")
}
