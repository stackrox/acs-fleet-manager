package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	"github.com/stretchr/testify/require"
)

func ReturningError() *errors.ServiceError {
	return nil
}

func TestAssignCluster(t *testing.T) {
	t.Setenv("RHACS_CLUSTER_MIGRATION", "true")
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	clusters := []*api.Cluster{
		testCluster("initial-cluster-1234"),
		testCluster("new-cluster-1234"),
	}

	helper, adminClient, teardown := test.NewAdminHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	orgID := "13640203"

	dummyRoutes := []dbapi.DataPlaneCentralRoute{
		{Domain: "test", Router: "test"},
		{Domain: "test", Router: "test"},
	}

	dummyRoutesJSON, err := json.Marshal(&dummyRoutes)
	require.NoError(t, err, "Unexpected error setting up test central routes")

	centrals := []*dbapi.CentralRequest{
		{
			MultiAZ:          clusters[0].MultiAZ,
			Owner:            "assigclusteruser1",
			Region:           clusters[0].Region,
			CloudProvider:    clusters[0].CloudProvider,
			Name:             "assign-cluster-central",
			OrganisationID:   orgID,
			Status:           constants2.CentralRequestStatusReady.String(),
			InstanceType:     clusters[0].SupportedInstanceType,
			ClusterID:        clusters[0].ClusterID,
			Meta:             api.Meta{ID: api.NewID()},
			RoutesCreated:    true,
			Routes:           dummyRoutesJSON,
			RoutesCreationID: "dummy-route-creation-id",
		},
		{
			MultiAZ:          clusters[0].MultiAZ,
			Owner:            "assigclusteruser2",
			Region:           clusters[0].Region,
			CloudProvider:    clusters[0].CloudProvider,
			Name:             "assign-cluster-central-2",
			OrganisationID:   orgID,
			Status:           constants2.CentralRequestStatusReady.String(),
			InstanceType:     clusters[0].SupportedInstanceType,
			ClusterID:        clusters[0].ClusterID,
			Meta:             api.Meta{ID: api.NewID()},
			RoutesCreated:    true,
			Routes:           dummyRoutesJSON,
			RoutesCreationID: "dummy-route-creation-id",
		},
	}

	db := test.TestServices.DBFactory.New()
	require.NoError(t, db.Create(&clusters).Error)
	require.NoError(t, db.Create(&centrals).Error)

	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedAdminContext(account, nil)

	res, err := adminClient.DefaultApi.AssignCentralCluster(ctx, centrals[0].Meta.ID, private.CentralAssignClusterRequest{ClusterId: clusters[1].ClusterID})
	if err != nil {
		if err, ok := err.(private.GenericOpenAPIError); ok {
			t.Fatal(string(err.Body()), res.StatusCode)
		}
	}

	cr, sErr := test.TestServices.DinosaurService.GetByID(centrals[0].Meta.ID)
	if sErr != nil {
		// not using require.NoError because serviceErr is a type wrapping
		// require would wrap it in an error interface which would cause it to fail on nil comparisons
		t.Fatal("Unexpected error getting central request", err)
	}

	require.Equal(t, "new-cluster-1234", cr.ClusterID, "ClusterID was not set properly.")
	require.False(t, cr.RoutesCreated, "RoutesCreated should be reset to false.")
	require.Nil(t, cr.Routes, "Stored Routes content should be nil.")
	require.Empty(t, cr.RoutesCreationID, "Stored RoutesCreationID should be reset to empty string")
	require.Equal(t, constants2.CentralRequestStatusProvisioning.String(), cr.Status, "Status should change from ready to provisioning.")
	require.True(t, cr.EnteredProvisioning.Valid, "EnteredProvisioning time should be valid")
	// can't require only after here as this might introduce a timing flake when this test runs through faster then
	// the precision of the stored time
	require.True(t, cr.CreatedAt.Equal(cr.EnteredProvisioning.Time) || cr.CreatedAt.After(cr.EnteredProvisioning.Time))
}

func TestAssignClusterCentralMismatch(t *testing.T) {
	t.Setenv("RHACS_CLUSTER_MIGRATION", "true")
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	clusters := []*api.Cluster{
		testCluster("initial-cluster-1234"),
		testCluster("new-cluster-1234"),
	}

	helper, adminClient, teardown := test.NewAdminHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	orgID := "13640203"

	centrals := []*dbapi.CentralRequest{
		{
			MultiAZ:        clusters[0].MultiAZ,
			Owner:          "assigclusteruser1",
			Region:         "non-matching-region",
			CloudProvider:  clusters[0].CloudProvider,
			Name:           "assign-cluster-central",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusReady.String(),
			InstanceType:   clusters[0].SupportedInstanceType,
			ClusterID:      clusters[0].ClusterID,
			Meta:           api.Meta{ID: api.NewID()},
		},
	}

	db := test.TestServices.DBFactory.New()
	require.NoError(t, db.Create(&clusters).Error)
	require.NoError(t, db.Create(&centrals).Error)

	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedAdminContext(account, nil)

	res, err := adminClient.DefaultApi.AssignCentralCluster(ctx, centrals[0].Meta.ID, private.CentralAssignClusterRequest{ClusterId: clusters[1].ClusterID})
	require.Error(t, err, "Expected bad requests error for central AssignCluster to non-matching region")
	require.NotNil(t, res)
	require.Equal(t, http.StatusBadRequest, res.StatusCode, "Expected bad request for central AssignCluster to non-matching region")
}

func testCluster(clusterID string) *api.Cluster {
	return &api.Cluster{
		CloudProvider:         "testprovider",
		Region:                "testregion",
		MultiAZ:               false,
		ClusterID:             clusterID,
		Status:                api.ClusterReady,
		ProviderType:          api.ClusterProviderStandalone,
		ClusterDNS:            "some.test.dns",
		SupportedInstanceType: "testtype",
		Schedulable:           true,
	}
}
