// Package fleetmanagerclient is a fmClientAuthWithOCMRefreshToken for the CLI to connect to the fleetmanager.
package fleetmanagerclient

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	impl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
)

var (
	singletonStaticTokenInstance         sync.Once
	fmAuthenticatedClientWithStaticToken *fleetmanager.Client

	fmClientAuthWithRHOASToken  *fleetmanager.Client
	singletonRHOASTokenInstance sync.Once
)

const (
	defaultFleetManagerEndpoint = "http://localhost:8000"
	fleetManagerEndpointEnvVar  = "FMCLI_FLEET_MANAGER_ENDPOINT"
	StaticTokenEnvVar           = "STATIC_TOKEN"
	rhoasTokenEnvVar            = "RHOAS_TOKEN"
)

// AuthenticatedClientWithRHOASToken returns a rest client for fleet-manager API using a static OCM token for authentication.
// This function should only be used for CLI commands.
func AuthenticatedClientWithRHOASToken(ctx context.Context) *fleetmanager.Client {
	rhoasToken := os.Getenv(rhoasTokenEnvVar)
	if rhoasToken == "" {
		panic(fmt.Sprintf("%s not set. Please set RHOAS token with 'export %s=<token>'", rhoasTokenEnvVar, rhoasTokenEnvVar))
	}

	fleetManagerEndpoint := os.Getenv(fleetManagerEndpointEnvVar)
	if fleetManagerEndpoint == "" {
		fleetManagerEndpoint = defaultFleetManagerEndpoint
	}

	singletonRHOASTokenInstance.Do(func() {
		auth, err := impl.NewAuth(ctx, impl.StaticTokenAuthName, impl.Option{
			Static: impl.StaticOption{
				StaticToken: rhoasToken,
			},
		})
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}

		fmClientAuthWithRHOASToken, err = impl.NewClient(fleetManagerEndpoint, auth)
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}
	})

	// sleep timer necessary to avoid "token issued in future" errors for time lags between fleet-manager running on a
	// local VM and the OCM server.
	if fleetManagerEndpoint == defaultFleetManagerEndpoint {
		time.Sleep(5 * time.Second)
	}
	return fmClientAuthWithRHOASToken
}

// AuthenticatedClientWithStaticToken returns a rest client to the fleet-manager and receives the static refresh token.
// This function will panic on an error, designed to be used by the fleet-manager CLI.
func AuthenticatedClientWithStaticToken(ctx context.Context) *fleetmanager.Client {
	staticToken := os.Getenv(StaticTokenEnvVar)
	if staticToken == "" {
		panic(fmt.Sprintf("%s not set. Please set OCM token with 'export %s=$(ocm token --refresh)'", StaticTokenEnvVar, StaticTokenEnvVar))
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
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}

		fmAuthenticatedClientWithStaticToken, err = impl.NewClient(fleetManagerEndpoint, auth)
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}
	})

	// sleep timer necessary to avoid "token issued in future" errors for time lags between fleet-manager running on a
	// local VM and the OCM server.
	if fleetManagerEndpoint == defaultFleetManagerEndpoint {
		time.Sleep(5 * time.Second)
	}
	return fmAuthenticatedClientWithStaticToken
}
