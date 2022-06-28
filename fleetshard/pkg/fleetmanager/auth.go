package fleetmanager

import (
	_ "embed"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strings"
)

// AuthType represents the supported authentication types for the client.
type AuthType int

const (

	// OCMTokenAuthType supports authentication via the refresh_token grant using an offline token provided by
	// console.redhat.com. The access token will be refreshed 1 minute before expiring via the refresh_token grant.
	OCMTokenAuthType AuthType = iota
	// RHSSOAuthType supports authentication via the client_credentials grant using client_id / client_secret via
	// sso.redhat.com. The access token will be refreshed 1 minute before expiring via the client_credentials grant.
	// NOTE: It can only be used in conjunction with the token-refresher.
	// See dp-terraform/helm/rhacs-terraform/templates/fleetshard-sync.yaml for more information.
	RHSSOAuthType
	// StaticTokenAuthType supports authentication via injecting a static token. The static token has static claims,
	// more details can be found within config/static-token-payload.json. The access token will not expire.
	StaticTokenAuthType
)

func (a AuthType) String() string {
	return [...]string{
		"OCM",
		"RHSSO",
	}[a]
}

// AuthTypeFromString will return the AuthType based on the string representation given. If no matching AuthType is
// found, an error will be returned.
func AuthTypeFromString(s string) (AuthType, error) {
	switch s {
	case RHSSOAuthType.String():
		return RHSSOAuthType, nil
	case OCMTokenAuthType.String():
		return OCMTokenAuthType, nil
	default:
		return OCMTokenAuthType, errors.Errorf("No valid auth type given, expected one of the following values"+
			" [%s] but got %q", strings.Join(getAllAuthTypes(), ","), s)
	}
}

// Auth will handle adding authentication information to HTTP requests.
type Auth interface {
	// AddAuth will add authentication information to the request, i.e. in the form of the Authorization header.
	AddAuth(req *http.Request) error
}

// NewAuth will return Auth that can be used to add authentication of a specific AuthType to be added to HTTP requests.
func NewAuth(t AuthType) (Auth, error) {
	switch t {
	case RHSSOAuthType:
		return newRHSSOAuth()
	case StaticTokenAuthType:
		return newStaticTokenAuth()
	default:
		return newOcmAuth()
	}
}

// setBearer is a helper to set a bearer token as authorization header on the http.Request.
func setBearer(req *http.Request, token string) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
}

// getAllAuthTypes is a helper used within logging to list the possible values for auth types.
func getAllAuthTypes() []string {
	return []string{
		RHSSOAuthType.String(),
		OCMTokenAuthType.String(),
	}
}
