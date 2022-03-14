package fleetshardsync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	coreTest "github.com/stackrox/acs-fleet-manager/test"
	"github.com/golang-jwt/jwt/v4"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// defaultUpdateDataplaneClusterStatusFunc - The default behaviour for updating data plane cluster status in each Fleetshard Sync reconcile.
// Retrieves all clusters in the database in a 'waiting_for_fleetshard_operator' state and updates it to 'ready' once all of the addons are installed.
var defaultUpdateDataplaneClusterStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error {
	var clusterService services.ClusterService
	var ocmConfig *ocm.OCMConfig
	var fleetshardConfig *config.FleetshardConfig
	helper.Env.MustResolveAll(&clusterService, &ocmConfig, &fleetshardConfig)
	clusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterWaitingForFleetShardOperator,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterWaitingForFleetShardOperator, err)
	}

	for _, cluster := range clusters {
		managedDinosaurAddon, err := ocmClient.GetAddon(cluster.ClusterID, ocmConfig.DinosaurOperatorAddonID)
		if err != nil {
			return err
		}

		fleetShardOperatorAddon, err := ocmClient.GetAddon(cluster.ClusterID, ocmConfig.FleetshardAddonID)
		if err != nil {
			return err
		}

		if managedDinosaurAddon.State() == clustersmgmtv1.AddOnInstallationStateReady && (fleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateReady || fleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateInstalling) {
			ctx := NewAuthenticatedContextForDataPlaneCluster(helper, cluster.ClusterID)
			clusterStatusUpdateRequest := SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
			if _, err := privateClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, cluster.ClusterID, *clusterStatusUpdateRequest); err != nil {
				return fmt.Errorf("failed to update cluster status via agent endpoint: %v", err)
			}
		}
	}
	return nil
}

// defaultUpdateDinosaurStatusFunc - The default behaviour for updating dinosaur status in each Fleetshard Sync reconcile.
// This function retrieves and updates Dinosaurs in all ready and full data plane clusters.
// Any Dinosaurs marked for deletion are updated to 'deleting'
// Dinosaurs with any other status are updated to 'ready'
var defaultUpdateDinosaurStatusFunc = func(helper *coreTest.Helper, privateClient *private.APIClient) error {
	var clusterService services.ClusterService
	helper.Env.MustResolveAll(&clusterService)

	var dataplaneClusters []*api.Cluster
	readyDataplaneClusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterReady,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterReady, err)
	}

	dataplaneClusters = append(dataplaneClusters, readyDataplaneClusters...)

	fullDataplaneClusters, err := clusterService.FindAllClusters(services.FindClusterCriteria{
		Status: api.ClusterFull,
	})
	if err != nil {
		return fmt.Errorf("Unable to retrieve a list of clusters in a '%s' state: %s", api.ClusterFull, err)
	}

	dataplaneClusters = append(dataplaneClusters, fullDataplaneClusters...)

	for _, dataplaneCluster := range dataplaneClusters {
		ctx := NewAuthenticatedContextForDataPlaneCluster(helper, dataplaneCluster.ClusterID)

		dinosaurList, _, err := privateClient.AgentClustersApi.GetDinosaurs(ctx, dataplaneCluster.ClusterID)
		if err != nil {
			return err
		}

		dinosaurStatusList := make(map[string]private.DataPlaneDinosaurStatus)
		for _, dinosaur := range dinosaurList.Items {
			id := dinosaur.Metadata.Annotations.MasId
			if dinosaur.Spec.Deleted {
				dinosaurStatusList[id] = GetDeletedDinosaurStatusResponse()
			} else {
				// Update any other clusters not in a 'deprovisioning' state to 'ready'
				dinosaurStatusList[id] = GetReadyDinosaurStatusResponse()
			}
		}

		if _, err = privateClient.AgentClustersApi.UpdateDinosaurClusterStatus(ctx, dataplaneCluster.ClusterID, dinosaurStatusList); err != nil {
			return err
		}
	}
	return nil
}

type MockFleetshardSyncBuilder interface {
	// SetUpdateDataplaneClusterStatusFunc - Sets behaviour for updating dataplane clusters in each Fleetshard sync reconcile
	SetUpdateDataplaneClusterStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error)
	// SetUpdateDinosaurStatusFunc - Sets behaviour for updating dinosaur clusters in each Fleetshard sync reconcile
	SetUpdateDinosaurStatusFunc(func(helper *coreTest.Helper, privateClient *private.APIClient) error)
	// SetInterval - Sets the repeat interval for the mock Fleetshard sync
	SetInterval(interval time.Duration)
	// Build - Builds a mock Fleetshard sync
	Build() MockFleetshardSync
}

// Mock Fleetshard Sync Builder
type mockFleetshardSyncBuilder struct {
	kfsync mockFleetshardSync
}

var _ MockFleetshardSyncBuilder = &mockFleetshardSyncBuilder{}

func NewMockFleetshardSyncBuilder(helper *coreTest.Helper, t *testing.T) MockFleetshardSyncBuilder {
	var ocmClient ocm.ClusterManagementClient
	helper.Env.MustResolveAll(&ocmClient)
	return &mockFleetshardSyncBuilder{
		kfsync: mockFleetshardSync{
			helper:                       helper,
			t:                            t,
			ocmClient:                    ocmClient,
			privateClient:                test.NewPrivateAPIClient(helper),
			updateDataplaneClusterStatus: defaultUpdateDataplaneClusterStatusFunc,
			updateDinosaurClusterStatus:  defaultUpdateDinosaurStatusFunc,
			interval:                     1 * time.Second,
		},
	}
}

func (m *mockFleetshardSyncBuilder) SetUpdateDataplaneClusterStatusFunc(updateDataplaneClusterStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error) {
	m.kfsync.updateDataplaneClusterStatus = updateDataplaneClusterStatusFunc
}

func (m *mockFleetshardSyncBuilder) SetUpdateDinosaurStatusFunc(updateDinosaurStatusFunc func(helper *coreTest.Helper, privateClient *private.APIClient) error) {
	m.kfsync.updateDinosaurClusterStatus = updateDinosaurStatusFunc
}

func (m *mockFleetshardSyncBuilder) SetInterval(interval time.Duration) {
	m.kfsync.interval = interval
}

func (m *mockFleetshardSyncBuilder) Build() MockFleetshardSync {
	return &m.kfsync
}

type MockFleetshardSync interface {
	// Start - Starts the Fleetshard sync reconciler
	Start()
	// Stop - Stops the Fleetshard sync reconciler
	Stop()
}

type mockFleetshardSync struct {
	helper                       *coreTest.Helper
	t                            *testing.T
	ocmClient                    ocm.Client
	ticker                       *time.Ticker
	privateClient                *private.APIClient
	interval                     time.Duration
	updateDataplaneClusterStatus func(helper *coreTest.Helper, privateClient *private.APIClient, ocmClient ocm.Client) error
	updateDinosaurClusterStatus  func(helper *coreTest.Helper, privateClient *private.APIClient) error
}

var _ MockFleetshardSync = &mockFleetshardSync{}

func (m *mockFleetshardSync) Start() {
	// starts reconcile immediately and then on every repeat interval
	// run reconcile
	m.t.Log("Starting mock fleetshard sync")
	m.ticker = time.NewTicker(m.interval)
	go func() {
		for range m.ticker.C {
			m.reconcileDataplaneClusters()
			m.reconcileDinosaurClusters()
		}
	}()
}

func (m *mockFleetshardSync) Stop() {
	m.ticker.Stop()
}

func (m *mockFleetshardSync) reconcileDataplaneClusters() {
	if err := m.updateDataplaneClusterStatus(m.helper, m.privateClient, m.ocmClient); err != nil {
		m.t.Logf("Unable to update dataplane cluster status: %s", err)
	}
}

func (m *mockFleetshardSync) reconcileDinosaurClusters() {
	if err := m.updateDinosaurClusterStatus(m.helper, m.privateClient); err != nil {
		m.t.Logf("Failed to update Dinosaur cluster status: %s", err)
	}
}

// Returns an authenticated context to be used for calling the data plane endpoints
func NewAuthenticatedContextForDataPlaneCluster(h *coreTest.Helper, clusterID string) context.Context {
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)

	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": keycloakConfig.DinosaurRealm.ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": {"fleetshard_operator"},
		},
		"fleetshard-operator-cluster-id": clusterID,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), private.ContextAccessToken, token)

	return ctx
}

// Returns a sample data plane cluster status request with available capacity
func SampleDataPlaneclusterStatusRequestWithAvailableCapacity() *private.DataPlaneClusterUpdateStatusRequest {
	return &private.DataPlaneClusterUpdateStatusRequest{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		DinosaurOperator: []private.DataPlaneClusterUpdateStatusRequestDinosaurOperator{
			{
				Ready:   true,
				Version: "dinosaur-operator.v0.23.0-0",
				DinosaurVersions: []string{
					"2.7.0",
					"2.5.3",
					"2.6.2",
				},
			},
			{
				Ready:   true,
				Version: "dinosaur-operator.v0.21.0-0",
				DinosaurVersions: []string{
					"2.7.0",
					"2.3.1",
					"2.1.2",
				},
			},
		},
	}
}

// Return a Dinosaur status for a deleted cluster
func GetDeletedDinosaurStatusResponse() private.DataPlaneDinosaurStatus {
	return private.DataPlaneDinosaurStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Deleted",
			},
		},
	}
}

func GetDefaultReportedDinosaurVersion() string {
	return "2.7.0"
}

func GetDefaultReportedDinosaurOperatorVersion() string {
	return "dinosaur-operator.v0.23.0-0"
}

// Return a dinosaur status for a ready cluster
func GetReadyDinosaurStatusResponse() private.DataPlaneDinosaurStatus {
	return private.DataPlaneDinosaurStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Versions: private.DataPlaneDinosaurStatusVersions{
			Dinosaur:         GetDefaultReportedDinosaurVersion(),
			DinosaurOperator: GetDefaultReportedDinosaurOperatorVersion(),
		},
	}
}

func GetErrorDinosaurStatusResponse() private.DataPlaneDinosaurStatus {
	return private.DataPlaneDinosaurStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Reason: "Error",
				Status: "False",
			},
		},
	}
}

func GetErrorWithCustomMessageDinosaurStatusResponse(message string) private.DataPlaneDinosaurStatus {
	res := GetErrorDinosaurStatusResponse()
	res.Conditions[0].Message = message
	return res
}
