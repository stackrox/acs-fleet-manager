// Package fleetmanagerclient is a client for the CLI to connect to the fleetmanager.
package fleetmanagerclient

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
)

var (
	singletonInstance sync.Once
	client            *fleetmanager.Client
)

const (
	defaultFleetManagerEndpoint = "http://localhost:8000"
	fleetManagerEndpointEnvVar  = "FMCLI_FLEET_MANAGER_ENDPOINT"
	ocmRefreshTokenEnvVar       = "OCM_TOKEN"
	staticTokenEnvVar           = "STATIC_TOKEN"
)

// AuthenticatedClientWithStaticToken returns a rest client to the fleet-manager using a static token.
func AuthenticatedClientWithStaticToken() *fleetmanager.Client {
	staticToken := os.Getenv(staticTokenEnvVar)
	if staticToken == "" {
		panic(fmt.Sprintf("%s not set. Please set static token with 'export %s=<token>'", staticTokenEnvVar, staticTokenEnvVar))
	}

	fleetManagerEndpoint := os.Getenv(fleetManagerEndpointEnvVar)
	if fleetManagerEndpoint == "" {
		fleetManagerEndpoint = defaultFleetManagerEndpoint
	}

	singletonInstance.Do(func() {
		auth, err := fleetmanager.NewAuth("STATIC_TOKEN", fleetmanager.Option{
			Static: fleetmanager.StaticOption{
				StaticToken: staticToken,
			},
		})
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}

		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
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
	return client

}

// AuthenticatedClientWithOCM returns a rest client to the fleet-manager and receives the OCM refresh token.
// This function will panic on an error, designed to be used by the fleet-manager CLI.
func AuthenticatedClientWithOCM() *fleetmanager.Client {
	ocmRefreshToken := os.Getenv(ocmRefreshTokenEnvVar)
	if ocmRefreshToken == "" {
		panic(fmt.Sprintf("%s not set. Please set OCM token with 'export %s=$(ocm token --refresh)'", ocmRefreshTokenEnvVar, ocmRefreshTokenEnvVar))
	}

	fleetManagerEndpoint := os.Getenv(fleetManagerEndpointEnvVar)
	if fleetManagerEndpoint == "" {
		fleetManagerEndpoint = defaultFleetManagerEndpoint
	}

	singletonInstance.Do(func() {
		auth, err := fleetmanager.NewAuth("OCM", fleetmanager.Option{
			Ocm: fleetmanager.OCMOption{
				RefreshToken: ocmRefreshToken,
			},
		})
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}

		client, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
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
	return client
}
