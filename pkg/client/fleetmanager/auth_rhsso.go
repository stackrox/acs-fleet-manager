package fleetmanager

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
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
	tokenFilePath string
}

type rhSSOAuthFactory struct{}

// GetName ...
func (f *rhSSOAuthFactory) GetName() string {
	return rhSSOAuthName
}

// CreateAuth ...
func (f *rhSSOAuthFactory) CreateAuth(o Option) (Auth, error) {
	tokenFilePath := o.sso.TokenFile
	if _, err := shared.ReadFile(tokenFilePath); err != nil {
		return nil, fmt.Errorf("reading token file: %w", err)
	}
	return &rhSSOAuth{
		tokenFilePath: tokenFilePath,
	}, nil
}

// AddAuth ...
func (r *rhSSOAuth) AddAuth(req *http.Request) error {
	// The file is populated by the token-refresher, which will ensure the token is not expired.
	token, err := shared.ReadFile(r.tokenFilePath)
	if err != nil {
		return errors.Wrapf(err, "reading token file %q within rhsso auth", r.tokenFilePath)
	}

	setBearer(req, token)
	return nil
}
