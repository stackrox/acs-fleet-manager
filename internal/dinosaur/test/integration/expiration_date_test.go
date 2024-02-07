package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	// TODO(ROX-10709) "github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"

	"github.com/antihax/optional"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services/quota"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers/dinosaurmgrs"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test/common"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"
	"github.com/stackrox/acs-fleet-manager/test/mocks"

	// TODO(ROX-10709)  "github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
)

// TestDinosaurExpiredAt validates the changing of the expired_at field.
func TestDinosaurExpirationManager(t *testing.T) {

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster("expiration-test-cluster", 1)})
	})
	defer teardown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	db := test.TestServices.DBFactory
	var id string

	// Create a central, wait for it to be accepted.
	{
		// POST responses per openapi spec: 201, 409, 500
		k := public.CentralRequestPayload{
			Region:        mocks.MockCluster.Region().ID(),
			CloudProvider: mocks.MockCluster.CloudProvider().ID(),
			Name:          "test-expiration-date",
			MultiAz:       testMultiAZ,
		}
		central, resp, err := common.WaitForDinosaurCreateToBeAccepted(ctx, db, client, k)
		id = central.Id
		defer client.DefaultApi.DeleteCentralById(ctx, central.Id, false)

		// dinosaur successfully registered with database
		Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
		Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
		Expect(central.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
		Expect(central.Kind).To(Equal(presenters.KindDinosaur))
		Expect(central.Href).To(Equal(fmt.Sprintf("/api/rhacs/v1/centrals/%s", central.Id)))
	}

	adminAPI := test.NewAdminPrivateAPIClient(h).DefaultApi

	central, _, err := adminAPI.GetCentralById(ctx, id)
	Expect(err).NotTo(HaveOccurred(), "Error getting central:  %v", err)
	Expect(central.ExpiredAt).To(BeNil())

	// Set expired_at.
	then := time.Now().Add(time.Hour)
	adminAPI.UpdateCentralExpiredAtById(ctx, central.Id, "test",
		&private.UpdateCentralExpiredAtByIdOpts{
			Timestamp: optional.NewString(then.Format(time.RFC3339)),
		})

	central, _, err = adminAPI.GetCentralById(ctx, id)
	Expect(err).NotTo(HaveOccurred(), "Error getting central:  %v", err)
	Expect(central.ExpiredAt).To(Equal(then))

	qmlc := quotamanagement.NewQuotaManagementListConfig()

	// Reset expired_at via expiration date manager.
	qs := quota.NewDefaultQuotaServiceFactory(test.TestServices.OCMClient, db, qmlc)
	cfg := config.NewCentralConfig()
	mgr := dinosaurmgrs.NewExpirationDateManager(test.TestServices.DinosaurService, qs, cfg)
	svcErrs := mgr.Reconcile()
	Expect(svcErrs).To(BeEmpty())

	// Check it is reset.
	central, _, err = adminAPI.GetCentralById(ctx, id)
	Expect(err).NotTo(HaveOccurred(), "Error getting central:  %v", err)
	Expect(central.ExpiredAt).To(BeNil())

	m := &services.QuotaServiceFactoryMock{
		GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
			return &services.QuotaServiceMock{
				HasQuotaAllowanceFunc: func(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
					return false, nil
				},
			}, nil
		},
	}

	// Disable quota.
	mgr = dinosaurmgrs.NewExpirationDateManager(test.TestServices.DinosaurService, m, cfg)
	svcErrs = mgr.Reconcile()
	Expect(svcErrs).To(BeEmpty())

	// Check the central is expired.
	central, _, err = adminAPI.GetCentralById(ctx, id)
	Expect(err).NotTo(HaveOccurred(), "Error getting central:  %v", err)
	Expect(central.ExpiredAt).ToNot(BeNil())
}
