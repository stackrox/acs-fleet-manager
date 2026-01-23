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
	"github.com/stackrox/acs-fleet-manager/internal/central"
	adminprivate "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/workers"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
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
	APIServer             *server.APIServer
	BootupServices        []environments.BootService
	CloudProvidersService services.CloudProvidersService
	ClusterService        services.ClusterService
	OCMClient             ocm.ClusterManagementClient
	OCMConfig             *ocm.OCMConfig
	CentralService        services.CentralService
	ClusterManager        *workers.ClusterManager
	ServerConfig          *server.ServerConfig
}

// TestServices ...
var TestServices Services

// NewCentralHelper Register a test
// This should be run before every integration test
func NewCentralHelper(t *testing.T, server *httptest.Server) (*test.Helper, *public.APIClient, func()) {
	return NewCentralHelperWithHooks(t, server, nil)
}

// NewCentralHelperWithHooks helper, public API Client and teardown function for integration testing public API endpoints
func NewCentralHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, *public.APIClient, func()) {
	h, teardown := newCentralHelperWithHooks(t, server, configurationHook)
	if err := h.Env.ServiceContainer.Resolve(&TestServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}
	return h, NewAPIClient(h), teardown
}

// NewAdminHelperWithHooks returns helper, adminprivate.APIClient and teardown function for integration testing Admin API endpoints
func NewAdminHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, *adminprivate.APIClient, func()) {
	h, teardown := newCentralHelperWithHooks(t, server, configurationHook)
	if err := h.Env.ServiceContainer.Resolve(&TestServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}
	var iamConfig *iam.IAMConfig
	h.Env.MustResolve(&iamConfig)
	if iamConfig == nil {
		glog.Fatal("Unable to resolve IAMConfig")
	}
	h.AuthHelper.OcmTokenIssuer = iamConfig.InternalSSORealm.ValidIssuerURI

	return h, NewAdminPrivateAPIClient(h), teardown
}

func newCentralHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, func()) {
	return test.NewHelperWithHooks(t, server, configurationHook, central.ConfigProviders(), di.ProvideValue(environments.BeforeCreateServicesHook{
		Func: func(dataplaneClusterConfig *config.DataplaneClusterConfig, centralConfig *config.CentralConfig, ocmConfig *ocm.OCMConfig) {
			centralConfig.CentralLifespan.EnableDeletionOfExpiredCentral = true
			dataplaneClusterConfig.DataPlaneClusterScalingType = config.NoScaling // disable scaling by default as it will be activated in specific tests
			// Integration tests require a valid OCM client. This requires OCM service account credentials to be set.
			ocmConfig.EnableMock = false
			ocmConfig.ReadFiles()
		},
	}))
}

// NewAPIClient ...
func NewAPIClient(helper *test.Helper) *public.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}

// NewAdminPrivateAPIClient ...
func NewAdminPrivateAPIClient(helper *test.Helper) *adminprivate.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := adminprivate.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := adminprivate.NewAPIClient(openapiConfig)
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
