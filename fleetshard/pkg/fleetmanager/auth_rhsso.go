package fleetmanager

import (
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"net/http"
)

type rhSSOAuth struct {
	tokenFilePath string
}

func newRHSSOAuth() (*rhSSOAuth, error) {
	cfg, err := config.Singleton()
	if err != nil {
		return nil, err
	}
	tokenFilePath := cfg.RHSSOTokenFilePath
	if _, err := shared.ReadFile(tokenFilePath); err != nil {
		return nil, err
	}
	return &rhSSOAuth{
		tokenFilePath: tokenFilePath,
	}, nil
}

func (r *rhSSOAuth) AddAuth(req *http.Request) error {
	// The file is populated by the token-refresher, which will ensure the token is not expired.
	token, err := shared.ReadFile(r.tokenFilePath)
	if err != nil {
		return errors.Wrapf(err, "reading token file %q within rhsso auth", r.tokenFilePath)
	}

	setBearer(req, token)
	return nil
}
