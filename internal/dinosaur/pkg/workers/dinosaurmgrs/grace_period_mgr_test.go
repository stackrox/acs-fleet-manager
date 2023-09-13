package dinosaurmgrs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

func TestGracePeriodManager(t *testing.T) {
	withEntitlement := func(e bool) (*services.QuotaServiceMock, *services.QuotaServiceFactoryMock) {
		qs := &services.QuotaServiceMock{
			IsQuotaEntitlementActiveFunc: func(dinosaur *dbapi.CentralRequest) (bool, error) {
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
			UpdateFunc: func(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
				return nil
			},
		}
	}
	defaultCfg := &config.CentralConfig{
		Quota: config.NewCentralQuotaConfig(),
	}

	t.Run("no centrals, no problem", func(t *testing.T) {
		dinoSvc := withCentrals()
		quotaSvc, quotaFactory := withEntitlement(true)
		gpm := NewGracePeriodManager(dinoSvc, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		assert.Len(t, dinoSvc.ListByStatusCalls(), 1)
		assert.Empty(t, dinoSvc.UpdateCalls())
		assert.Empty(t, quotaSvc.IsQuotaEntitlementActiveCalls())
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("unset grace", func(t *testing.T) {
		now := time.Now()
		central := &dbapi.CentralRequest{GraceFrom: &now}
		dinoSvc := withCentrals(central)
		quotaSvc, quotaFactory := withEntitlement(true)
		gpm := NewGracePeriodManager(dinoSvc, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		assert.Nil(t, central.GraceFrom)
		assert.Len(t, dinoSvc.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.IsQuotaEntitlementActiveCalls(), 1)
		assert.Len(t, dinoSvc.UpdateCalls(), 1)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("set grace", func(t *testing.T) {
		now := time.Now()
		central := &dbapi.CentralRequest{}
		dinoSvc := withCentrals(central)
		quotaSvc, quotaFactory := withEntitlement(false)
		gpm := NewGracePeriodManager(dinoSvc, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		require.NotNil(t, central.GraceFrom)
		assert.Less(t, now, *central.GraceFrom)
		assert.Len(t, dinoSvc.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.IsQuotaEntitlementActiveCalls(), 1)
		assert.Len(t, dinoSvc.UpdateCalls(), 1)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("quota cost cache in use", func(t *testing.T) {
		now := time.Now()
		centralA := &dbapi.CentralRequest{GraceFrom: &now, OrganisationID: "one"}
		centralB := &dbapi.CentralRequest{GraceFrom: &now, OrganisationID: "one"}
		centralC := &dbapi.CentralRequest{GraceFrom: &now, OrganisationID: "another"}
		dinoSvc := withCentrals(centralA, centralB, centralC)
		quotaSvc, quotaFactory := withEntitlement(true)
		gpm := NewGracePeriodManager(dinoSvc, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		assert.Nil(t, centralA.GraceFrom)
		assert.Nil(t, centralB.GraceFrom)
		assert.Len(t, dinoSvc.ListByStatusCalls(), 1)
		assert.Len(t, quotaSvc.IsQuotaEntitlementActiveCalls(), 2)
		assert.Len(t, dinoSvc.UpdateCalls(), 3)
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})

	t.Run("skip deleting and deprovision statuses", func(t *testing.T) {
		centralA := &dbapi.CentralRequest{Status: constants.CentralRequestStatusDeleting.String()}
		centralB := &dbapi.CentralRequest{Status: constants.CentralRequestStatusDeprovision.String()}
		dinoSvc := withCentrals(centralA, centralB)
		quotaSvc, quotaFactory := withEntitlement(false)
		gpm := NewGracePeriodManager(dinoSvc, quotaFactory, defaultCfg)
		errs := gpm.Reconcile()
		require.Empty(t, errs)
		assert.Nil(t, centralA.GraceFrom)
		assert.Nil(t, centralB.GraceFrom)
		assert.Len(t, dinoSvc.ListByStatusCalls(), 1)
		assert.Empty(t, quotaSvc.IsQuotaEntitlementActiveCalls())
		assert.Empty(t, dinoSvc.UpdateCalls())
		assert.Len(t, quotaFactory.GetQuotaServiceCalls(), 1)
	})
}
