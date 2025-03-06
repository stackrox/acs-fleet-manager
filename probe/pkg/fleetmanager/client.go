// Package fleetmanager ...
package fleetmanager

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
	"github.com/stackrox/acs-fleet-manager/probe/config"
)

// New creates a new fleet manager client.
func New(ctx context.Context, config config.Config) (fleetmanager.PublicAPI, error) {
	auth, err := impl.NewAuth(ctx, config.AuthType, impl.OptionFromEnv())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager authentication")
	}

	client, err := impl.NewClient(config.FleetManagerEndpoint, auth, impl.WithUserAgent("fleet-manager-probe-service"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager client")
	}

	return client.PublicAPI(), nil
}
