package envtokenauth

import (
	"fmt"
	"net/http"
	"os"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
)

// Implements the Auth interface for simple static token based authentication
// while fetching the token from a custom environment variable.
type envTokenAuth struct {
	token string
}

// CreateAuth creates a new Auth instance which implements static token authentication
// while fetching the token from the environment using the specified environment variable name.
func CreateAuth(name string) (fleetmanager.Auth, error) {
	token := os.Getenv(name)
	if token == "" {
		return nil, fmt.Errorf("no token named %q found in current environment", name)
	}
	return &envTokenAuth{
		token: token,
	}, nil
}

// AddAuth adds an Authorization header to the provided HTTP request.
func (a *envTokenAuth) AddAuth(req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	return nil
}
