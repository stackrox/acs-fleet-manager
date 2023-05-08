// Package fleetmanagerclient is a fmClientAuthWithOCMRefreshToken for the CLI to connect to the fleetmanager.
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
	singletonOCMRefreshTokenInstance sync.Once
	fmClientAuthWithOCMRefreshToken  *fleetmanager.Client

	fmClientAuthWithRHOASToken  *fleetmanager.Client
	singletonRHOASTokenInstance sync.Once
)

const (
	defaultFleetManagerEndpoint = "http://localhost:8000"
	fleetManagerEndpointEnvVar  = "FMCLI_FLEET_MANAGER_ENDPOINT"
	ocmRefreshTokenEnvVar       = "OCM_TOKEN"
	rhoasTokenEnvVar            = "RHOAS_TOKEN"
)

// AuthenticatedClientWithRHOASToken returns a rest fmClientAuthWithOCMRefreshToken to the fleet-manager using a static token.
// This function should only be used for CLI commands.
func AuthenticatedClientWithRHOASToken() *fleetmanager.Client {
	rhoasToken := os.Getenv(rhoasTokenEnvVar)
	if rhoasToken == "" {
		panic(fmt.Sprintf("%s not set. Please set RHOAS token with 'export %s=<token>'", rhoasTokenEnvVar, rhoasTokenEnvVar))
	}

	fleetManagerEndpoint := os.Getenv(fleetManagerEndpointEnvVar)
	if fleetManagerEndpoint == "" {
		fleetManagerEndpoint = defaultFleetManagerEndpoint
	}

	singletonRHOASTokenInstance.Do(func() {
		auth, err := fleetmanager.NewAuth(fleetmanager.StaticTokenAuthName, fleetmanager.Option{
			Static: fleetmanager.StaticOption{
				StaticToken: rhoasToken,
			},
		})
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}

		fmClientAuthWithRHOASToken, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
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

// AuthenticatedClientWithOCM returns a rest fmClientAuthWithOCMRefreshToken to the fleet-manager and receives the OCM refresh token.
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

	singletonOCMRefreshTokenInstance.Do(func() {
		auth, err := fleetmanager.NewAuth(fleetmanager.OCMAuthName, fleetmanager.Option{
			Ocm: fleetmanager.OCMOption{
				RefreshToken: ocmRefreshToken,
			},
		})
		if err != nil {
			glog.Fatalf("Failed to create connection: %s", err)
			return
		}

		fmClientAuthWithOCMRefreshToken, err = fleetmanager.NewClient(fleetManagerEndpoint, auth)
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
	return fmClientAuthWithOCMRefreshToken
}
