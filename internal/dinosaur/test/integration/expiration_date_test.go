package integration

import (
	"testing"
	"time"

	// TODO(ROX-10709) "github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"

	"github.com/antihax/optional"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services/quota"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/workers/dinosaurmgrs"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
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
	h, _, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster("expiration-test-cluster", 1)})
	})
	defer teardown()

	// setup pre-requisites to performing requests
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss":    issuerURL,
		"org_id": orgID,
	}
	ctx := h.NewAuthenticatedContext(account, claims)

	dbFactory := test.TestServices.DBFactory
	var id string

	// Register a central in the DB
	{
		dinosaurCloudProvider := "dummy"
		// this value is taken from config/quota-management-list-configuration.yaml
		orgID := "13640203"
		db := test.TestServices.DBFactory.New()

		dinosaurs := []*dbapi.CentralRequest{{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur",
			OrganisationID: orgID,
			Status:         constants.CentralRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		}}
		if err := db.Create(&dinosaurs).Error; err != nil {
			Expect(err).NotTo(HaveOccurred())
			return
		}
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
	qs := quota.NewDefaultQuotaServiceFactory(test.TestServices.OCMClient, dbFactory, qmlc)
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
