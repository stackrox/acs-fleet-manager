package integration

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
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

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	cluster1 := test.NewMockDataplaneCluster("initial-cluster", 5)
	cluster1.ClusterID = "initial-cluster-1234"
	cluster2 := test.NewMockDataplaneCluster("new-cluster", 5)
	cluster2.ClusterID = "new-cluster-1234"
	cluster2.CloudProvider = cluster1.CloudProvider

	helper, adminClient, teardown := test.NewAdminHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{cluster1, cluster2})
	})
	defer teardown()

	orgID := "13640203"

	centrals := []*dbapi.CentralRequest{
		{
			MultiAZ:        false,
			Owner:          "assigclusteruser1",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  cluster1.CloudProvider,
			Name:           "assign-cluster-central",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusReady.String(),
			InstanceType:   types.STANDARD.String(),
			ClusterID:      cluster1.ClusterID,
			Meta:           api.Meta{ID: api.NewID()},
		},
		{
			MultiAZ:        false,
			Owner:          "assigclusteruser2",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  cluster1.CloudProvider,
			Name:           "assign-cluster-central-2",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusReady.String(),
			InstanceType:   types.STANDARD.String(),
			ClusterID:      cluster1.ClusterID,
			Meta:           api.Meta{ID: api.NewID()},
		},
	}

	db := test.TestServices.DBFactory.New()
	require.NoError(t, db.Create(&centrals).Error)

	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account, jwt.MapClaims{
		"realm_access": map[string]interface{}{
			"roles": []string{"acs-fleet-manager-admin-full"},
		},
	})

	_, err := adminClient.DefaultApi.AssignCentralCluster(ctx, centrals[0].Meta.ID, private.CentralAssignClusterRequest{ClusterId: cluster2.ClusterID})
	require.NoError(t, err)
}
