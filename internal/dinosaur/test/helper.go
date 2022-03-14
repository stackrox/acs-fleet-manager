package test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur"
	adminprivate "github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/workers"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/client/observatorium"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	coreWorkers "github.com/stackrox/acs-fleet-manager/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/test"
	"github.com/goava/di"
	"github.com/golang/glog"
)

type Services struct {
	di.Inject
	DBFactory             *db.ConnectionFactory
	KeycloakConfig        *keycloak.KeycloakConfig
	DinosaurConfig        *config.DinosaurConfig
	MetricsServer         *server.MetricsServer
	HealthCheckServer     *server.HealthCheckServer
	Workers               []coreWorkers.Worker
	LeaderElectionManager *coreWorkers.LeaderElectionManager
	APIServer             *server.ApiServer
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

var TestServices Services

// Register a test
// This should be run before every integration test
func NewDinosaurHelper(t *testing.T, server *httptest.Server) (*test.Helper, *public.APIClient, func()) {
	return NewDinosaurHelperWithHooks(t, server, nil)
}

func NewDinosaurHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, *public.APIClient, func()) {
	h, teardown := test.NewHelperWithHooks(t, server, configurationHook, dinosaur.ConfigProviders(), di.ProvideValue(environments.BeforeCreateServicesHook{
		Func: func(dataplaneClusterConfig *config.DataplaneClusterConfig, dinosaurConfig *config.DinosaurConfig, observabilityConfiguration *observatorium.ObservabilityConfiguration, fleetshardConfig *config.FleetshardConfig) {
			dinosaurConfig.DinosaurLifespan.EnableDeletionOfExpiredDinosaur = true
			observabilityConfiguration.EnableMock = true
			dataplaneClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
			dataplaneClusterConfig.RawKubernetesConfig = nil                      // disable applying resources for standalone clusters
		},
	}))
	if err := h.Env.ServiceContainer.Resolve(&TestServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}
	return h, NewApiClient(h), teardown
}

func NewApiClient(helper *test.Helper) *public.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}

func NewPrivateAPIClient(helper *test.Helper) *private.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := private.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := private.NewAPIClient(openapiConfig)
	return client
}

func NewAdminPrivateAPIClient(helper *test.Helper) *adminprivate.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := adminprivate.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := adminprivate.NewAPIClient(openapiConfig)
	return client
}

func NewMockDataplaneCluster(name string, capacity int) config.ManualCluster {
	return config.ManualCluster{
		Name:                  name,
		CloudProvider:         mocks.MockCluster.CloudProvider().ID(),
		Region:                mocks.MockCluster.Region().ID(),
		MultiAZ:               true,
		Schedulable:           true,
		DinosaurInstanceLimit: capacity,
		Status:                api.ClusterReady,
		SupportedInstanceType: "eval,standard",
	}
}
