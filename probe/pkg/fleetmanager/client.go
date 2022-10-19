package fleetmanager

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
)

const (
	authType = "RHSSO"
	// StandardInstanceType denotes an instance with quota.
	StandardInstanceType = "standard"
)

// Client is an interface for the fleetmanager client.
type Client interface {
	CreateCentral(context.Context, bool, public.CentralRequestPayload) (public.CentralRequest, *http.Response, error)
	DeleteCentralById(context.Context, string, bool) (*http.Response, error)
	GetCentralById(context.Context, string) (public.CentralRequest, *http.Response, error)
}

// New creates a new fleet manager client.
func New(config *config.Config) (Client, error) {
	auth, err := fleetmanager.NewAuth(authType, fleetmanager.Option{
		Sso: fleetmanager.RHSSOOption{
			ClientID:     config.RHSSOClientID,
			ClientSecret: config.RHSSOClientSecret, // pragma: allowlist secret
			Realm:        config.RHSSORealm,
			Endpoint:     config.RHSSOEndpoint,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create fleet manager authentication")
	}

	client, err := fleetmanager.NewClient(config.FleetManagerEndpoint, auth, fleetmanager.WithUserAgent("probe-service"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create fleet manager client")
	}

	return client.PublicAPI(), nil
}
