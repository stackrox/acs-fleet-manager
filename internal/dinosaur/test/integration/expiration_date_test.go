package integration

import (
	"fmt"
	"testing"
	"time"

	// TODO(ROX-10709) "github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"

	"github.com/antihax/optional"
	"github.com/golang-jwt/jwt/v4"
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
	token := h.CreateJWTStringWithClaim(account, claims)
	privateConfig := private.NewConfiguration()
	privateConfig.BasePath = fmt.Sprintf("http://%s", test.TestServices.ServerConfig.BindAddress)
	privateConfig.DefaultHeader = map[string]string{
		"Authorization": "Bearer " + token,
	}
	adminAPI := private.NewAPIClient(privateConfig).DefaultApi
	request, _, err := adminAPI.CreateCentral(ctx, false, private.CentralRequestPayload{
		CloudProvider: "dummy",
		MultiAz:       false,
		Name:          "dummy-dinosaur",
		Region:        mocks.MockCluster.Region().ID(),
	})
	Expect(err).NotTo(HaveOccurred(), "Error creating central:  %v", err)

	id := request.Id
	defer adminAPI.DeleteCentralById(ctx, id, false)

	getCentral := func() private.Central {
		central, _, err := adminAPI.GetCentralById(ctx, id)
		Expect(err).NotTo(HaveOccurred(), "Error getting central:  %v", err)
		return central
	}

	Expect(getCentral().ExpiredAt).To(BeNil())

	// Set expired_at.
	then := time.Now().Add(time.Hour)
	adminAPI.UpdateCentralExpiredAtById(ctx, id, "test",
		&private.UpdateCentralExpiredAtByIdOpts{
			Timestamp: optional.NewString(then.Format(time.RFC3339)),
		})

	Expect(getCentral().ExpiredAt).To(Equal(then))

	qmlc := quotamanagement.NewQuotaManagementListConfig()

	quotaServiceFactory := quota.NewDefaultQuotaServiceFactory(test.TestServices.OCMClient, test.TestServices.DBFactory, qmlc)

	// Reset expired_at via expiration date manager.
	var centralConfig *config.CentralConfig
	h.Env.ServiceContainer.Resolve(&centralConfig)
	mgr := dinosaurmgrs.NewExpirationDateManager(test.TestServices.DinosaurService, quotaServiceFactory, centralConfig)
	svcErrs := mgr.Reconcile()
	Expect(svcErrs).To(BeEmpty())

	// Check it is reset.
	Expect(getCentral().ExpiredAt).To(BeNil())

	quotaMock := &services.QuotaServiceFactoryMock{
		GetQuotaServiceFunc: func(quotaType api.QuotaType) (services.QuotaService, *errors.ServiceError) {
			return &services.QuotaServiceMock{
				HasQuotaAllowanceFunc: func(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
					return false, nil
				},
			}, nil
		},
	}

	// Disable quota.
	mgr = dinosaurmgrs.NewExpirationDateManager(test.TestServices.DinosaurService, quotaMock, centralConfig)
	svcErrs = mgr.Reconcile()
	Expect(svcErrs).To(BeEmpty())

	// Check the central is expired.
	Expect(getCentral().ExpiredAt).ToNot(BeNil())
}
