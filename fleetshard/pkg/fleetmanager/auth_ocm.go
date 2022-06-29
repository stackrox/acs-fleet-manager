package fleetmanager

import (
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"time"
)

const (
	ocmTokenExpirationMargin = time.Minute
	ocmClientID              = "cloud-services"
)

type ocmAuth struct {
	conn *sdk.Connection
}

func newOcmAuth() (*ocmAuth, error) {
	initialToken := os.Getenv("OCM_TOKEN")
	if initialToken == "" {
		return nil, errors.New("empty ocm token")
	}

	l, err := sdk.NewGlogLoggerBuilder().Build()
	if err != nil {
		return nil, err
	}

	builder := sdk.NewConnectionBuilder().
		Client(ocmClientID, "").
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
	access, _, err := o.conn.TokensContext(req.Context(), ocmTokenExpirationMargin)
	if err != nil {
		return errors.Wrap(err, "retrieving access token via OCM auth type")
	}

	setBearer(req, access)
	return nil
}
