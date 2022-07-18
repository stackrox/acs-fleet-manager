package fleetmanager

import (
	"net/http"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
)

const (
	ocmTokenExpirationMargin = time.Minute
	ocmClientID              = "cloud-services"
	ocmAuthName              = "OCM"
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

func (f *ocmAuthFactory) GetName() string {
	return ocmAuthName
}

func (f *ocmAuthFactory) CreateAuth() (Auth, error) {
	cfg, err := config.Singleton()
	if err != nil {
		return nil, err
	}
	initialToken := cfg.OCMRefreshToken
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
