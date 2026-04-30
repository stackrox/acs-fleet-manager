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
	threeYearsAgo := now.Add(-3 * 365 * 24 * time.Hour)
	oneYearAgo := now.Add(-365 * 24 * time.Hour)
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

	centralStandardOld := newDeletedCentral("standard-old", false)
	centralStandardRecent := newDeletedCentral("standard-recent", false)
	centralInternalOld := newDeletedCentral("internal-old", true)
	centralInternalRecent := newDeletedCentral("internal-recent", true)
	centralActive := &dbapi.CentralRequest{
		Meta:          api.Meta{ID: api.NewID()},
		Name:          "active",
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "test-user",
		Status:        constants.CentralRequestStatusReady.String(),
	}

	for _, c := range []*dbapi.CentralRequest{centralStandardOld, centralStandardRecent, centralInternalOld, centralInternalRecent, centralActive} {
		require.NoError(t, db.Create(c).Error)
	}

	setSoftDeleted(t, db, centralStandardOld.ID, threeYearsAgo)
	setSoftDeleted(t, db, centralStandardRecent.ID, oneYearAgo)
	setSoftDeleted(t, db, centralInternalOld.ID, thirtyDaysAgo)
	setSoftDeleted(t, db, centralInternalRecent.ID, sevenDaysAgo)

	errs := mgr.Reconcile()
	require.Empty(t, errs)

	assert.True(t, isHardDeleted(t, db, centralStandardOld.ID), "standard central deleted 3y ago should be pruned")
	assert.False(t, isHardDeleted(t, db, centralStandardRecent.ID), "standard central deleted 1y ago should be kept")
	assert.True(t, isHardDeleted(t, db, centralInternalOld.ID), "internal central deleted 30d ago should be pruned")
	assert.False(t, isHardDeleted(t, db, centralInternalRecent.ID), "internal central deleted 7d ago should be kept")
	assert.False(t, isHardDeleted(t, db, centralActive.ID), "active central should never be pruned")

	// Verify idempotency: running again should produce no errors.
	errs = mgr.Reconcile()
	require.Empty(t, errs)
}

func newDeletedCentral(name string, internal bool) *dbapi.CentralRequest {
	return &dbapi.CentralRequest{
		Meta:          api.Meta{ID: api.NewID()},
		Name:          name,
		Region:        "us-east-1",
		CloudProvider: "aws",
		Owner:         "test-user",
		Status:        constants.CentralRequestStatusDeleting.String(),
		Internal:      internal,
	}
}

func setSoftDeleted(t *testing.T, db *gorm.DB, id string, deletedAt time.Time) {
	t.Helper()
	err := db.Unscoped().
		Model(&dbapi.CentralRequest{}).
		Where("id = ?", id).
		Update("deleted_at", deletedAt).Error
	require.NoError(t, err)
}

func isHardDeleted(t *testing.T, db *gorm.DB, id string) bool {
	t.Helper()
	var count int64
	err := db.Unscoped().Model(&dbapi.CentralRequest{}).Where("id = ?", id).Count(&count).Error
	require.NoError(t, err)
	return count == 0
}
