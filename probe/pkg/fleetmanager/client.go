package fleetmanager

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
)

// Client is an interface for the fleetmanager client.
//
//go:generate moq -out client_moq.go . Client
type Client interface {
	CreateCentral(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error)
	DeleteCentralById(ctx context.Context, id string, async bool) (*http.Response, error)
	GetCentralById(ctx context.Context, id string) (public.CentralRequest, *http.Response, error)
	GetCentrals(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error)
}

// New creates a new fleet manager client.
func New(config *config.Config) (Client, error) {
	auth, err := fleetmanager.NewRHSSOAuth(fleetmanager.RHSSOOption{
		ClientID:     config.RHSSOClientID,
		ClientSecret: config.RHSSOClientSecret, // pragma: allowlist secret
		Realm:        config.RHSSORealm,
		Endpoint:     config.RHSSOEndpoint,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager authentication")
	}

	client, err := fleetmanager.NewClient(config.FleetManagerEndpoint, auth, fleetmanager.WithUserAgent("probe-service"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager client")
	}

	return client.PublicAPI(), nil
}
