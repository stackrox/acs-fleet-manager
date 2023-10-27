// Package test ...
package test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"

	"github.com/goava/di"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	coreWorkers "github.com/stackrox/acs-fleet-manager/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/test"
)

// Services ...
type Services struct {
	di.Inject
	DBFactory             *db.ConnectionFactory
	IAMConfig             *iam.IAMConfig
	CentralConfig         *config.CentralConfig
	MetricsServer         *server.MetricsServer
	HealthCheckServer     *server.HealthCheckServer
	Workers               []coreWorkers.Worker
	LeaderElectionManager *coreWorkers.LeaderElectionManager
	APIServer             *server.APIServer
	BootupServices        []environments.BootService
	CloudProvidersService services.CloudProvidersService
	ClusterService        services.ClusterService
	OCMClient             ocm.ClusterManagementClient
	OCMConfig             *ocm.OCMConfig
	DinosaurService       services.DinosaurService
	ObservatoriumClient   *observatorium.Client
	ClusterManager        *workers.ClusterManager
	ServerConfig          *server.ServerConfig
}

// TestServices ...
var TestServices Services

// NewDinosaurHelper Register a test
// This should be run before every integration test
func NewDinosaurHelper(t *testing.T, server *httptest.Server) (*test.Helper, *public.APIClient, func()) {
	return NewDinosaurHelperWithHooks(t, server, nil)
}

// NewDinosaurHelperWithHooks ...
func NewDinosaurHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, *public.APIClient, func()) {
	dpConfig := config.SingletonDataplaneClusterConfig()
	dpConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
	dpConfig.RawKubernetesConfig = nil                      // disable applying resources for standalone clusters

	centralConfig := config.GetCentralConfig()
	centralConfig.CentralLifespan.EnableDeletionOfExpiredCentral = true
	observabilityConfig := observatorium.GetObservabilityConfigurationConfig()
	observabilityConfig.EnableMock = true

	h, teardown := test.NewHelperWithHooks(t, server, configurationHook, dinosaur.ConfigProviders())
	if err := h.Env.ServiceContainer.Resolve(&TestServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}
	return h, NewAPIClient(), teardown
}

// NewAPIClient ...
func NewAPIClient() *public.APIClient {
	serverConfig := server.GetServerConfig()
	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}

// NewMockDataplaneCluster ...
func NewMockDataplaneCluster(name string, capacity int) config.ManualCluster {
	return config.ManualCluster{
		Name:                  name,
		CloudProvider:         mocks.MockCluster.CloudProvider().ID(),
		Region:                mocks.MockCluster.Region().ID(),
		MultiAZ:               true,
		Schedulable:           true,
		CentralInstanceLimit:  capacity,
		Status:                api.ClusterReady,
		SupportedInstanceType: "eval,standard",
	}
}
