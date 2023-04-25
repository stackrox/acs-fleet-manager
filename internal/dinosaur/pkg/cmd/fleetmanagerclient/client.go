// Package fleetmanagerclient is a client for the CLI to connect to the fleetmanager.
package fleetmanagerclient

import (
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
)

// AuthenticatedClientWithOCM returns a rest client to the fleet-manager and receives the OCM refresh token.
// This function will panic on an error, designed to be used by the fleet-manager CLI.
func AuthenticatedClientWithOCM() *fleetmanager.Client {
	ocmRefreshToken := os.Getenv(ocmRefreshTokenEnvVar)
	if ocmRefreshToken == "" {
		panic("OCM_TOKEN not set. Please set OCM token with 'export OCM_TOKEN=$(ocm token --refresh)'")
	}

	fleetmanagerEndpoint := defaultFleetManagerEndpoint
	if os.Getenv(fleetManagerEndpointEnvVar) != "" {
		fleetmanagerEndpoint = os.Getenv(fleetManagerEndpointEnvVar)
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

		client, err = fleetmanager.NewClient(fleetmanagerEndpoint, auth)
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}
	})

	// sleep timer necessary to avoid "token issued in future" errors for time lags between fleet-manager running on a
	// local VM and the OCM server.
	if fleetmanagerEndpoint == defaultFleetManagerEndpoint {
		time.Sleep(5 * time.Second)
	}
	return client
}
