package dinosaurmgrs

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

func TestExpirationDateManager(t *testing.T) {
	withEntitlement := func(e bool) (*services.QuotaServiceMock, *services.QuotaServiceFactoryMock) {
		qs := &services.QuotaServiceMock{
			HasQuotaAllowanceFunc: func(central *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
				return e, nil
			},
		}
		return qs, &services.QuotaServiceFactoryMock{
			GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
				return qs, nil
			},
		}
	}
	withCentrals := func(centrals ...*dbapi.CentralRequest) *services.DinosaurServiceMock {
		return &services.DinosaurServiceMock{
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
	defaultCfg := &config.CentralConfig{
		Quota: config.NewCentralQuotaConfig(),
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
		assert.Nil(t, central.ExpiredAt)
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
		assert.Nil(t, centralA.ExpiredAt)
		assert.Nil(t, centralB.ExpiredAt)
		assert.Nil(t, centralC.ExpiredAt)
		assert.Nil(t, centralD.ExpiredAt)
		assert.Nil(t, centralE.ExpiredAt)
		assert.Len(t, centralService.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.HasQuotaAllowanceCalls(), 3)
		assert.Len(t, centralService.UpdatesCalls(), 5)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})
}
