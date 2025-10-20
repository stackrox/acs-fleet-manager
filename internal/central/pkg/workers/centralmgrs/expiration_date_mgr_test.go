package centralmgrs

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

const internalCentralID = "internal-central-id"

func TestExpirationDateManager(t *testing.T) {
	withEntitlement := func(e bool) (*services.QuotaServiceMock, *services.QuotaServiceFactoryMock) {
		qs := &services.QuotaServiceMock{
			HasQuotaAllowanceFunc: func(central *dbapi.CentralRequest, instanceType types.CentralInstanceType) (bool, *errors.ServiceError) {
				return e, nil
			},
		}
		return qs, &services.QuotaServiceFactoryMock{
			GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
				return qs, nil
			},
		}
	}
	withCentrals := func(centrals ...*dbapi.CentralRequest) *services.CentralServiceMock {
		return &services.CentralServiceMock{
			ListByStatusFunc: func(status ...constants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError) {
				return centrals, nil
			},
			UpdatesFunc: func(centralRequest *dbapi.CentralRequest, fields map[string]any) *errors.ServiceError {
				if _, ok := fields["expired_at"]; !ok {
					return errors.GeneralError("bad fields")
				}
				return nil
			},
		}
	}
	quotaConf := config.NewCentralQuotaConfig()
	quotaConf.InternalCentralIDs = []string{internalCentralID}
	defaultCfg := &config.CentralConfig{
		Quota: quotaConf,
	}

	t.Run("no centrals, no problem", func(t *testing.T) {
		centralService := withCentrals()
		quotaSvc, quotaFactory := withEntitlement(true)
		mgr := NewExpirationDateManager(centralService, quotaFactory, defaultCfg)
		errs := mgr.Reconcile()
		require.Empty(t, errs)
		assert.Len(t, centralService.ListByStatusCalls(), 1)
		assert.Empty(t, centralService.UpdatesCalls())
		assert.Empty(t, quotaSvc.HasQuotaAllowanceCalls())
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("unset expired_at", func(t *testing.T) {
		now := time.Now()
		central := &dbapi.CentralRequest{ExpiredAt: sql.NullTime{Time: now, Valid: true}}
		centralService := withCentrals(central)
		quotaSvc, quotaFactory := withEntitlement(true)
		gpm := NewExpirationDateManager(centralService, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		assert.False(t, central.ExpiredAt.Valid)
		assert.Len(t, centralService.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.HasQuotaAllowanceCalls(), 1)
		assert.Len(t, centralService.UpdatesCalls(), 1)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("set expired_at", func(t *testing.T) {
		now := time.Now()
		central := &dbapi.CentralRequest{}
		centralService := withCentrals(central)
		quotaSvc, quotaFactory := withEntitlement(false)
		gpm := NewExpirationDateManager(centralService, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		require.True(t, central.ExpiredAt.Valid)
		assert.Less(t, now, *dbapi.NullTimeToTimePtr(central.ExpiredAt))
		assert.Len(t, centralService.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.HasQuotaAllowanceCalls(), 1)
		assert.Len(t, centralService.UpdatesCalls(), 1)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("skip setting expired_at for internal central even if no valid quota", func(t *testing.T) {
		central := &dbapi.CentralRequest{}
		central.ID = internalCentralID
		centralService := withCentrals(central)
		quotaSvc, quotaFactory := withEntitlement(true)
		gpm := NewExpirationDateManager(centralService, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		require.False(t, central.ExpiredAt.Valid)
		assert.Len(t, quotaSvc.HasQuotaAllowanceCalls(), 0)
		assert.Len(t, centralService.UpdatesCalls(), 0)
	})

	t.Run("quota cost cache in use", func(t *testing.T) {
		now := sql.NullTime{Time: time.Now(), Valid: true}
		centralA := &dbapi.CentralRequest{ExpiredAt: now, OrganisationID: "one"}
		centralB := &dbapi.CentralRequest{ExpiredAt: now, OrganisationID: "one"}
		centralC := &dbapi.CentralRequest{ExpiredAt: now, OrganisationID: "another"}
		centralD := &dbapi.CentralRequest{ExpiredAt: now, OrganisationID: "another"}
		centralE := &dbapi.CentralRequest{ExpiredAt: now, OrganisationID: "another", CloudAccountID: "Zeus"}
		centralService := withCentrals(centralA, centralB, centralC, centralD, centralE)
		quotaSvc, quotaFactory := withEntitlement(true)
		gpm := NewExpirationDateManager(centralService, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		assert.False(t, centralA.ExpiredAt.Valid)
		assert.False(t, centralB.ExpiredAt.Valid)
		assert.False(t, centralC.ExpiredAt.Valid)
		assert.False(t, centralD.ExpiredAt.Valid)
		assert.False(t, centralE.ExpiredAt.Valid)
		assert.Len(t, centralService.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.HasQuotaAllowanceCalls(), 3)
		assert.Len(t, centralService.UpdatesCalls(), 5)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})
}
