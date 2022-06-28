package fleetmanager

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/golang/glog"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"net/http"
	"os"
	"strings"
	"time"
)

// AuthType represents the supported authentication types for the client.
type AuthType int

const (
	// OCMTokenAuthType supports authentication via the refresh_token grant using an offline token provided by
	// console.redhat.com. The access token will be refreshed 1 minute before expiring via the refresh_token grant.
	OCMTokenAuthType AuthType = iota
)

func (a AuthType) String() string {
	return [...]string{
		"OCM",
	}[a]
}

// AuthTypeFromString will return the AuthType based on the string representation given. If no matching AuthType is
// found, the issue will be logged and StaticTokenAuthType will be used as default.
func AuthTypeFromString(s string) AuthType {
	switch s {
	case OCMTokenAuthType.String():
		return OCMTokenAuthType
	default:
		glog.Warningf("No valid auth type given, expected one of the following values [%s] but got %q. "+
			"Defaulting to auth type %s",
			strings.Join(getAllAuthTypes(), ","), s, OCMTokenAuthType.String())
		return OCMTokenAuthType
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
	default:
		return newOcmAuth(os.Getenv("OCM_TOKEN"))
	}
}

type ocmAuth struct {
	conn *sdk.Connection
}

func newOcmAuth(initialToken string) (*ocmAuth, error) {
	if initialToken == "" {
		return nil, errors.New("empty ocm token")
	}

	l, err := sdk.NewGlogLoggerBuilder().Build()
	if err != nil {
		return nil, err
	}

	builder := sdk.NewConnectionBuilder().
		Client("cloud-services", "").
		Tokens(initialToken).
		Logger(l)

	// Check if the connection can be established and tokens can be retrieved.
	conn, err := builder.Build()
	if err != nil {
		return nil, err
	}
	_, _, err = conn.Tokens()
	if err != nil {
		return nil, err
	}

	return &ocmAuth{
		conn: conn,
	}, nil
}

func (o *ocmAuth) AddAuth(req *http.Request) error {
	// This will only do an external request iff the current access token of the connection has an expiration time
	// lower than 1 minute.
	access, _, err := o.conn.TokensContext(req.Context(), 1*time.Minute)
	if err != nil {
		return err
	}

	setBearer(req, access)
	return nil
}

type noAuth struct{}

func (n noAuth) AddAuth(_ *http.Request) error {
	return nil
}

// setBearer is a helper to set a bearer token as authorization header on the http.Request.
func setBearer(req *http.Request, token string) {
	// Do not attempt to modify any existing authorization headers.
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
}

// getAllAuthTypes is a helper used within logging to list the possible values for auth types.
func getAllAuthTypes() []string {
	return []string{
		OCMTokenAuthType.String(),
	}
}
