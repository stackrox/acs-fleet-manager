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

// GetName ...
func (f *staticTokenAuthFactory) GetName() string {
	return staticTokenAuthName
}

// CreateAuth ...
func (f *staticTokenAuthFactory) CreateAuth(o option) (Auth, error) {
	staticToken := o.Static.StaticToken
	if staticToken == "" {
		return nil, errors.New("no static token set")
	}
	return &staticTokenAuth{
		token: staticToken,
	}, nil
}

// AddAuth ...
func (s *staticTokenAuth) AddAuth(req *http.Request) error {
	setBearer(req, s.token)
	return nil
}
