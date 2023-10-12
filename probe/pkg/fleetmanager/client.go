// Package fleetmanager ...
package fleetmanager

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
)

// New creates a new fleet manager client.
func New(ctx context.Context, config *config.Config) (fleetmanager.PublicAPI, error) {
	auth, err := fleetmanager.NewAuth(ctx, config.AuthType, fleetmanager.OptionFromEnv())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager authentication")
	}

	client, err := fleetmanager.NewClient(config.FleetManagerEndpoint, auth, fleetmanager.WithUserAgent("fleet-manager-probe-service"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager client")
	}

	return client.PublicAPI(), nil
}
