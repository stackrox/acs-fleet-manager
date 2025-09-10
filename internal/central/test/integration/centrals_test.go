package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	// TODO(ROX-10709) "github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/test"
	"github.com/stackrox/acs-fleet-manager/internal/central/test/common"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"

	// TODO(ROX-10709)  "github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
)

const (
	mockCentralName = "test-central1"
	testMultiAZ     = true
)

// TestCentralCreate_Success validates the happy path of the central post endpoint:
func TestCentralCreate_Success(t *testing.T) {
	skipNotFullyImplementedYet(t)

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewCentralHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockCentralClusterName, 1)})
	})
	defer teardown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// POST responses per openapi spec: 201, 409, 500
	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockCentralName,
		MultiAz:       testMultiAZ,
	}

	central, resp, err := common.WaitForCentralCreateToBeAccepted(ctx, test.TestServices.DBFactory, client, k)

	// central successfully registered with database
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(central.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(central.Kind).To(Equal(presenters.KindCentral))
	Expect(central.Href).To(Equal(fmt.Sprintf("/api/rhacs/v1/centrals/%s", central.Id)))
}

func TestCentralCreate_TooManyCentrals(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.NewCentralHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockCentralClusterName, 1)})
	})
	defer tearDown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	centralCloudProvider := "dummy"
	// this value is taken from config/quota-management-list-configuration.yaml
	orgID := "13640203"

	// create dummy centrals
	db := test.TestServices.DBFactory.New()
	centrals := []*dbapi.CentralRequest{
		{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  centralCloudProvider,
			Name:           "dummy-central",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "dummyuser2",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  centralCloudProvider,
			Name:           "dummy-central-2",
			OrganisationID: orgID,
			Status:         constants2.CentralRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
	}

	if err := db.Create(&centrals).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockCentralName,
		MultiAz:       testMultiAZ,
	}

	_, resp, err := client.DefaultApi.CreateCentral(ctx, true, k)

	Expect(err).To(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestCentral_Delete(t *testing.T) {
	owner := "test-user"

	orgID := "13640203"

	sampleCentralID := api.NewID()

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, tearDown := test.NewCentralHelper(t, ocmServer)
	defer tearDown()

	userAccount := h.NewAccount(owner, "test-user", "test@gmail.com", orgID)

	userCtx := h.NewAuthenticatedContext(userAccount, nil)

	type args struct {
		ctx       context.Context
		centralID string
		async     bool
	}
	tests := []struct {
		name           string
		args           args
		verifyResponse func(resp *http.Response, err error)
	}{
		{
			name: "should fail when deleting central without async set to true",
			args: args{
				ctx:       userCtx,
				centralID: sampleCentralID,
				async:     false,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should fail when deleting central with empty id",
			args: args{
				ctx:       userCtx,
				centralID: "",
				async:     true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).NotTo(BeNil())
			},
		},
		{
			name: "should succeed when deleting central with valid id and context",
			args: args{
				ctx:       userCtx,
				centralID: sampleCentralID,
				async:     true,
			},
			verifyResponse: func(resp *http.Response, err error) {
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			},
		},
	}

	db := test.TestServices.DBFactory.New()
	// create a dummy cluster and assign a central to it
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

	// create a central that will be updated
	central := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: sampleCentralID,
		},
		MultiAZ:        true,
		Owner:          owner,
		Region:         "test",
		CloudProvider:  "test",
		Name:           "test-central",
		OrganisationID: orgID,
		Status:         constants.CentralRequestStatusReady.String(),
		ClusterID:      cluster.ClusterID,
		InstanceType:   types.EVAL.String(),
	}

	if err := db.Create(central).Error; err != nil {
		t.Errorf("failed to create Central db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewAPIClient(h)
			resp, err := client.DefaultApi.DeleteCentralById(tt.args.ctx, tt.args.centralID, tt.args.async)
			tt.verifyResponse(resp, err)
		})
	}
}
