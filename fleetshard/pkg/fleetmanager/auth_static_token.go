package fleetmanager

import (
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"net/http"
)

type staticTokenAuth struct {
	token string
}

func newStaticTokenAuth() (*staticTokenAuth, error) {
	cfg, err := config.Singleton()
	if err != nil {
		return nil, err
	}
	staticToken := cfg.StaticToken
	if staticToken == "" {
		return nil, errors.New("no static token set")
	}
	return &staticTokenAuth{
		token: staticToken,
	}, nil
}

func (s *staticTokenAuth) AddAuth(req *http.Request) error {
	setBearer(req, s.token)
	return nil
}
