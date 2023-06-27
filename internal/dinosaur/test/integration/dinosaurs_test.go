package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	// TODO(ROX-10709) "github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test/common"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"

	// TODO(ROX-10709)  "github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
)

const (
	mockDinosaurName = "test-dinosaur1"
	testMultiAZ      = true
)

// TestDinosaurCreate_Success validates the happy path of the dinosaur post endpoint:
func TestDinosaurCreate_Success(t *testing.T) {
	skipNotFullyImplementedYet(t)

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer teardown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// POST responses per openapi spec: 201, 409, 500
	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	dinosaur, resp, err := common.WaitForDinosaurCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)

	// dinosaur successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(dinosaur.Kind).To(Equal(presenters.KindDinosaur))
	Expect(dinosaur.Href).To(Equal(fmt.Sprintf("/api/rhacs/v1/centrals/%s", dinosaur.Id)))
}

func TestDinosaurCreate_TooManyDinosaurs(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer tearDown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	dinosaurCloudProvider := "dummy"
	// this value is taken from config/quota-management-list-configuration.yaml
	orgID := "13640203"

	// create dummy dinosaurs
	db := test.TestServices.DBFactory.New()
	dinosaurs := []*dbapi.CentralRequest{
		{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "dummyuser2",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur-2",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	_, resp, err := client.DefaultApi.CreateCentral(ctx, true, k)

	Expect(err).To(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestDinosaur_Delete(t *testing.T) {
	owner := "test-user"

	orgID := "13640203"

	sampleDinosaurID := api.NewID()

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewDinosaurHelper(t, ocmServer)
	defer tearDown()

	userAccount := h.NewAccount(owner, "test-user", "test@gmail.com", orgID)

	userCtx := h.NewAuthenticatedContext(userAccount, nil)

	type args struct {
		ctx        context.Context
		dinosaurID string
		async      bool
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting dinosaur without async set to true",
			args: args{
				ctx:        userCtx,
				dinosaurID: sampleDinosaurID,
				async:      false,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when deleting dinosaur with empty id",
			args: args{
				ctx:        userCtx,
				dinosaurID: "",
				async:      true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should succeed when deleting dinosaur with valid id and context",
			args: args{
				ctx:        userCtx,
				dinosaurID: sampleDinosaurID,
				async:      true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a dinosaur to it
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: api.NewID(),
		},
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             "baremetal",
		CloudProvider:      "baremetal",
		Status:             api.ClusterReady,
		IdentityProviderID: "some-id",
		ClusterDNS:         "some-cluster-dns",
		ProviderType:       api.ClusterProviderStandalone,
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Error("failed to create dummy cluster")
		return
	}

	// create a dinosaur that will be updated
	dinosaur := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID,
		},
		MultiAZ:        true,
		Owner:          owner,
		Region:         "test",
		CloudProvider:  "test",
		Name:           "test-dinosaur",
		OrganisationID: orgID,
		Status:         constants.CentralRequestStatusReady.String(),
		ClusterID:      cluster.ClusterID,
		InstanceType:   types.EVAL.String(),
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewAPIClient(h)
			resp, err := client.DefaultApi.DeleteCentralById(tt.args.ctx, tt.args.dinosaurID, tt.args.async)
			tt.verifyResponse(resp, err)
		})
	}
}
