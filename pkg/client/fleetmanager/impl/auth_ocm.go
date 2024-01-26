package impl

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
)

const (
	ocmTokenExpirationMargin = 5 * time.Minute
	ocmClientID              = "cloud-services"
	// OCMAuthName is the name of the OCM auth authentication method
	OCMAuthName = "OCM"
)

var (
	_          authFactory = (*ocmAuthFactory)(nil)
	_          Auth        = (*ocmAuth)(nil)
	ocmFactory             = &ocmAuthFactory{}
)

type ocmAuth struct {
	conn *sdk.Connection
}

type ocmAuthFactory struct{}

// GetName gets the name of the factory.
func (f *ocmAuthFactory) GetName() string {
	return OCMAuthName
}

// CreateAuth ...
func (f *ocmAuthFactory) CreateAuth(ctx context.Context, o Option) (Auth, error) {
	initialToken := o.Ocm.RefreshToken
	if initialToken == "" {
		return nil, errors.New("empty ocm token")
	}

	builder := sdk.NewConnectionBuilder().
		Client(ocmClientID, "").
		Tokens(initialToken)

	if o.Ocm.EnableLogger {
		l, err := sdk.NewGlogLoggerBuilder().Build()
		if err != nil {
			return nil, fmt.Errorf("creating Glog logger: %w", err)
		}
		builder.Logger(l)
	}

	// Check if the connection can be established and tokens can be retrieved.
	conn, err := builder.BuildContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating connection: %w", err)
	}
	_, _, err = conn.TokensContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving tokens: %w", err)
	}

	return &ocmAuth{
		conn: conn,
	}, nil
}

// AddAuth add auth token to the request retrieved from OCM.
func (o *ocmAuth) AddAuth(req *http.Request) error {
	// This will only do an external request iff the current access token of the connection has an expiration time
	// lower than 5 minutes.
	access, _, err := o.conn.TokensContext(req.Context(), ocmTokenExpirationMargin)
	if err != nil {
		return errors.Wrap(err, "retrieving access token via OCM auth type")
	}

	setBearer(req, access)
	return nil
}
