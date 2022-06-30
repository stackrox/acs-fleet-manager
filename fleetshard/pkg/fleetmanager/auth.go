package fleetmanager

import (
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

// Auth will handle adding authentication information to HTTP requests.
type Auth interface {
	// AddAuth will add authentication information to the request, i.e. in the form of the Authorization header.
	AddAuth(req *http.Request) error
}

type authFactory interface {
	GetName() string
	CreateAuth() (Auth, error)
}

var authFactoryRegistry map[string]authFactory

func init() {
	authFactoryRegistry = map[string]authFactory{
		ocmFactory.GetName():         ocmFactory,
		rhSSOFactory.GetName():       rhSSOFactory,
		staticTokenFactory.GetName(): staticTokenFactory,
	}
}

// NewAuth will return Auth that can be used to add authentication of a specific AuthType to be added to HTTP requests.
func NewAuth(t string) (Auth, error) {
	factory, exists := authFactoryRegistry[t]
	if !exists {
		return nil, errors.Errorf("invalid auth type found: %q, must be one of [%s]",
			t, strings.Join(getAllAuthTypes(), ","))
	}
	return factory.CreateAuth()
}

// setBearer is a helper to set a bearer token as authorization header on the http.Request.
func setBearer(req *http.Request, token string) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
}

// getAllAuthTypes is a helper used within logging to list the possible values for auth types.
func getAllAuthTypes() []string {
	authTypes := make([]string, 0, len(authFactoryRegistry))
	for authType := range authFactoryRegistry {
		authTypes = append(authTypes, authType)
	}
	return authTypes
}
