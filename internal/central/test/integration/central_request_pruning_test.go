package integration

import (
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/workers/centralmgrs"
	"github.com/stackrox/acs-fleet-manager/internal/central/test"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCentralRequestPruning(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewCentralHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	db := test.TestServices.DBFactory.New()
	mgr := centralmgrs.NewCentralRequestPruningManager(test.TestServices.DBFactory)

	now := time.Now()

	cases := []struct {
		name       string
		central    *dbapi.CentralRequest
		wantPruned bool
	}{
		{"standard central deleted 3y ago should be pruned", newDeletedCentral("standard-old", false, now.Add(-3*365*24*time.Hour)), true},
		{"standard central deleted 1y ago should be kept", newDeletedCentral("standard-recent", false, now.Add(-365*24*time.Hour)), false},
		{"internal central deleted 30d ago should be pruned", newDeletedCentral("internal-old", true, now.Add(-30*24*time.Hour)), true},
		{"internal central deleted 7d ago should be kept", newDeletedCentral("internal-recent", true, now.Add(-7*24*time.Hour)), false},
		{"active central should never be pruned", newActiveCentral(), false},
	}

	for _, tc := range cases {
		require.NoError(t, db.Unscoped().Create(tc.central).Error)
	}

	errs := mgr.Reconcile()
	require.Empty(t, errs)

	for _, tc := range cases {
		assert.Equal(t, tc.wantPruned, isHardDeleted(t, db, tc.central.ID), tc.name)
	}

	// Verify idempotency: running again should produce no errors.
	errs = mgr.Reconcile()
	require.Empty(t, errs)
}

func newDeletedCentral(name string, internal bool, deletedAt time.Time) *dbapi.CentralRequest {
	return &dbapi.CentralRequest{
		Meta: api.Meta{
			ID:        api.NewID(),
			DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true},
		},
		Name:          name,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "test-user",
		Status:        constants.CentralRequestStatusDeleting.String(),
		Internal:      internal,
	}
}

func newActiveCentral() *dbapi.CentralRequest {
	return &dbapi.CentralRequest{
		Meta:          api.Meta{ID: api.NewID()},
		Name:          "active",
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "test-user",
		Status:        constants.CentralRequestStatusReady.String(),
	}
}

func isHardDeleted(t *testing.T, db *gorm.DB, id string) bool {
	t.Helper()
	var count int64
	err := db.Unscoped().Model(&dbapi.CentralRequest{}).Where("id = ?", id).Count(&count).Error
	require.NoError(t, err)
	return count == 0
}
