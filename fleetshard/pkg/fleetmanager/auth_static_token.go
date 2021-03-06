package fleetmanager

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
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
func (f *staticTokenAuthFactory) CreateAuth() (Auth, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("creating the config singleton: %w", err)
	}
	staticToken := cfg.StaticToken
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
