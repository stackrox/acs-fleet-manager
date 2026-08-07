// Package fleetmanagerclient provides helpers for CLI commands to obtain fleet-manager API clients.
package fleetmanagerclient

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	impl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
)

type contextKey int

const clientKey contextKey = iota

// NewContext returns a context carrying the given fleet-manager client.
func NewContext(ctx context.Context, c *fleetmanager.Client) context.Context {
	return context.WithValue(ctx, clientKey, c)
}

// ClientFromContext returns the fleet-manager client stored in ctx.
// It panics if no client is found.
func ClientFromContext(ctx context.Context) *fleetmanager.Client {
	c, ok := ctx.Value(clientKey).(*fleetmanager.Client)
	if !ok || c == nil {
		panic("fleet-manager client not found in context; this command must run under 'admin'")
	}
	return c
}

var (
	singletonStaticTokenInstance         sync.Once
	fmAuthenticatedClientWithStaticToken *fleetmanager.Client
)

const (
	defaultFleetManagerEndpoint = "http://localhost:8000"
	fleetManagerEndpointEnvVar  = "FMCLI_FLEET_MANAGER_ENDPOINT"
	StaticTokenEnvVar           = "STATIC_TOKEN"
)

// AuthenticatedClientWithStaticToken returns a rest client to the fleet-manager and receives the static token.
// This function will panic on an error, designed to be used by the fleet-manager CLI.
func AuthenticatedClientWithStaticToken(ctx context.Context) *fleetmanager.Client {
	staticToken := os.Getenv(StaticTokenEnvVar)
	if staticToken == "" {
		panic(fmt.Sprintf("%s not set. Please set static token with 'export %s=<token>'", StaticTokenEnvVar, StaticTokenEnvVar))
	}

	fleetManagerEndpoint := os.Getenv(fleetManagerEndpointEnvVar)
	if fleetManagerEndpoint == "" {
		fleetManagerEndpoint = defaultFleetManagerEndpoint
	}

	singletonStaticTokenInstance.Do(func() {
		auth, err := impl.NewAuth(ctx, impl.StaticTokenAuthName, impl.Option{
			Static: impl.StaticOption{
				StaticToken: staticToken,
			},
		})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to create connection: %s\n", err)
			os.Exit(1)
		}

		fmAuthenticatedClientWithStaticToken, err = impl.NewClient(fleetManagerEndpoint, auth)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to create connection: %s\n", err)
			os.Exit(1)
		}
	})

	// sleep timer necessary to avoid "token issued in future" errors for time lags between fleet-manager running on a
	// local VM and the OCM server.
	if fleetManagerEndpoint == defaultFleetManagerEndpoint {
		time.Sleep(5 * time.Second)
	}
	return fmAuthenticatedClientWithStaticToken
}
