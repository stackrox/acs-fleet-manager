package fleetmanager

import (
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"net/http"
	"os"
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
	if _, err := os.Stat(tokenFilePath); err != nil {
		return nil, err
	}
	return &rhSSOAuth{
		tokenFilePath: tokenFilePath,
	}, nil
}

func (r *rhSSOAuth) AddAuth(req *http.Request) error {
	// The file is populated by the token-refresher, which will ensure the token is not expired.
	contents, err := os.ReadFile(r.tokenFilePath)
	if err != nil {
		return err
	}

	setBearer(req, string(contents))
	return nil
}
