package fleetmanager

import (
	"net/http"

	"github.com/pkg/errors"
)

const (
	staticTokenAuthName = "STATIC_TOKEN"
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
	return staticTokenAuthName
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
