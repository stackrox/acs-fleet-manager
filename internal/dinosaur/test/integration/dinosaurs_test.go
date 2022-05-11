package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	// TODO(ROX-10709) "github.com/stackrox/acs-fleet-manager/pkg/quota_management"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test/common"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test/mocks/fleetshardsync"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	// TODO(ROX-10709)  "github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
)

const (
	mockDinosaurName    = "test-dinosaur1"
	testMultiAZ         = true
	// TODO(ROX-10709) invalidDinosaurName = "Test_Cluster9"
	// TODO(ROX-10709) longDinosaurName    = "thisisaninvaliddinosaurclusternamethatexceedsthenamesizelimit"
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

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

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
	 // TODO(ROX-10709) Expect(dinosaur.InstanceType).To(Equal(types.STANDARD.String()))
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

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	dinosaurCloudProvider := "dummy"
	// this value is taken from config/quota-management-list-configuration.yaml
	orgId := "13640203"

	// create dummy dinosaurs
	db := test.TestServices.DBFactory.New()
	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          "dummyuser1",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
			InstanceType:   types.STANDARD.String(),
		},
		{
			MultiAZ:        false,
			Owner:          "dummyuser2",
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  dinosaurCloudProvider,
			Name:           "dummy-dinosaur-2",
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusAccepted.String(),
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

// TestDinosaurPost_Validations tests the API validations performed by the dinosaur creation endpoint
//
// these could also be unit tests
/* TODO(ROX-10709)
func TestDinosaurPost_Validations(t *testing.T) {
	skipNotFullyImplementedYet(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	tests := []struct {
		name     string
		body     public.CentralRequestPayload
		wantCode int
	}{
		{
			name: "HTTP 400 when region not supported",
			body: public.CentralRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        "us-east-3",
				Name:          mockDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when provider not supported",
			body: public.CentralRequestPayload{
				MultiAz:       mocks.MockCluster.MultiAZ(),
				CloudProvider: "azure",
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when MultiAZ false provided",
			body: public.CentralRequestPayload{
				MultiAz:       false,
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          mockDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name not provided",
			body: public.CentralRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is not valid",
			body: public.CentralRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          invalidDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "HTTP 400 when name is too long",
			body: public.CentralRequestPayload{
				CloudProvider: mocks.MockCluster.CloudProvider().ID(),
				MultiAz:       mocks.MockCluster.MultiAZ(),
				Region:        mocks.MockCluster.Region().ID(),
				Name:          longDinosaurName,
			},
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			_, resp, _ := client.DefaultApi.CreateCentral(ctx, true, tt.body)
			Expect(resp.StatusCode).To(Equal(tt.wantCode))
		})
	}
}
*/

/* TODO(ROX-10709)
// TestDinosaurPost_NameUniquenessValidations tests dinosaur cluster name uniqueness verification during its creation by the API
func TestDinosaurPost_NameUniquenessValidations(t *testing.T) {
	skipNotFullyImplementedYet(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 3)})
	})
	defer tearDown()

	// create two random accounts in same organisation
	account1 := h.NewRandAccount()
	ctx1 := h.NewAuthenticatedContext(account1, nil)

	account2 := h.NewRandAccount()
	ctx2 := h.NewAuthenticatedContext(account2, nil)

	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"

	// create a third account in a different organisation
	accountFromAnotherOrg := h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx3 := h.NewAuthenticatedContext(accountFromAnotherOrg, nil)

	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	// create the first dinosaur
	_, resp1, _ := client.DefaultApi.CreateCentral(ctx1, true, k)

	// attempt to create the second dinosaur with same name
	_, resp2, _ := client.DefaultApi.CreateCentral(ctx2, true, k)

	// create another dinosaur with same name for different org
	_, resp3, _ := client.DefaultApi.CreateCentral(ctx3, true, k)

	// verify that the first and third requests were accepted
	Expect(resp1.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp3.StatusCode).To(Equal(http.StatusAccepted))

	// verify that the second request resulted into an error accepted
	Expect(resp2.StatusCode).To(Equal(http.StatusConflict))

}
*/

/* TODO(ROX-10709)
// TestDinosaurGet tests getting dinosaurs via the API endpoint
func TestDinosaurGet(t *testing.T) {
	skipNotFullyImplementedYet(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	seedDinosaur, _, err := client.DefaultApi.CreateCentral(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded dinosaur request: %s", err.Error())
	}

	// 200 OK
	dinosaur, resp, err := client.DefaultApi.GetCentralById(ctx, seedDinosaur.Id)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to get dinosaur request:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(dinosaur.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(dinosaur.Kind).To(Equal(presenters.KindDinosaur))
	Expect(dinosaur.Href).To(Equal(fmt.Sprintf("/api/dinosaurs_mgmt/v1/dinosaurs/%s", dinosaur.Id)))
	Expect(dinosaur.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(dinosaur.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(dinosaur.Name).To(Equal(mockDinosaurName))
	Expect(dinosaur.Status).To(Equal(constants2.DinosaurRequestStatusAccepted.String()))
	// When dinosaur is in 'Accepted' state it means that it still has not been
	// allocated to a cluster, which means that fleetshard-sync has not reported
	// yet any status, so the version attribute (actual version) at this point
	// should still be empty
	Expect(dinosaur.Version).To(Equal(""))

	// 404 Not Found
	dinosaur, resp, _ = client.DefaultApi.GetCentralById(ctx, fmt.Sprintf("not-%s", seedDinosaur.Id))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// A different account than the one used to create the Dinosaur instance, within
	// the same organization than the used account used to create the Dinosaur instance,
	// should be able to read the Dinosaur cluster
	account = h.NewRandAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	dinosaur, _, _ = client.DefaultApi.GetCentralById(ctx, seedDinosaur.Id)
	Expect(dinosaur.Id).NotTo(BeEmpty())

	// An account in a different organization than the one used to create the
	// Dinosaur instance should get a 404 Not Found.
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "12147054"
	account = h.NewAccountWithNameAndOrg(faker.Name(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetCentralById(ctx, seedDinosaur.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// An allowed serviceaccount in config/quota-management-list-configuration.yaml
	// without an organization ID should get a 401 Unauthorized
	account = h.NewAllowedServiceAccount()
	ctx = h.NewAuthenticatedContext(account, nil)
	_, resp, _ = client.DefaultApi.GetCentralById(ctx, seedDinosaur.Id)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}
*/

func TestDinosaur_Delete(t *testing.T) {
	owner := "test-user"

	orgId := "13640203"

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

	userAccount := h.NewAccount(owner, "test-user", "test@gmail.com", orgId)

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
	dinosaur := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: sampleDinosaurID,
		},
		MultiAZ:                        true,
		Owner:                          owner,
		Region:                         "test",
		CloudProvider:                  "test",
		Name:                           "test-dinosaur",
		OrganisationId:                 orgId,
		Status:                         constants.DinosaurRequestStatusReady.String(),
		ClusterID:                      cluster.ClusterID,
		ActualDinosaurVersion:          "2.6.0",
		DesiredDinosaurVersion:         "2.6.0",
		ActualDinosaurOperatorVersion:  "v.23.0",
		DesiredDinosaurOperatorVersion: "v0.23.0",
		InstanceType:                   types.EVAL.String(),
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := test.NewApiClient(h)
			resp, err := client.DefaultApi.DeleteCentralById(tt.args.ctx, tt.args.dinosaurID, tt.args.async)
			tt.verifyResponse(resp, err)
		})
	}
}

/* TODO(ROX-10709)
// TestDinosaurList_Success tests getting dinosaur requests list
func TestDinosaurList_Success(t *testing.T) {
	skipNotFullyImplementedYet(t)

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(acl *quota_management.QuotaManagementListConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 1)})
	})
	defer teardown()

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()
	defer mockFleetshardSync.Stop()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	initCtx := h.NewAuthenticatedContext(account, nil)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.GetCentrals(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(initList.Items).To(BeEmpty(), "Expected empty dinosaur requests list")
	Expect(initList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(initList.Total).To(Equal(int32(0)), "Expected Total == 0")

	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	// POST dinosaur request to populate the list
	seedDinosaur, _, err := client.DefaultApi.CreateCentral(initCtx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded DinosaurRequest: %s", err.Error())
	}

	// get populated list of dinosaur requests
	afterPostList, _, err := client.DefaultApi.GetCentrals(initCtx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(afterPostList.Items)).To(Equal(1), "Expected dinosaur requests list length to be 1")
	Expect(afterPostList.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(afterPostList.Total).To(Equal(int32(1)), "Expected Total == 1")

	// get dinosaur request item from the list
	listItem := afterPostList.Items[0]

	// check whether the seedDinosaur properties are the same as those from the dinosaur request list item
	Expect(seedDinosaur.Id).To(Equal(listItem.Id))
	Expect(seedDinosaur.Kind).To(Equal(listItem.Kind))
	Expect(listItem.Kind).To(Equal(presenters.KindDinosaur))
	Expect(seedDinosaur.Href).To(Equal(listItem.Href))
	Expect(seedDinosaur.Region).To(Equal(listItem.Region))
	Expect(listItem.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(seedDinosaur.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(seedDinosaur.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockDinosaurName))
	Expect(listItem.Status).To(Equal(constants2.DinosaurRequestStatusAccepted.String()))

	// new account setup to prove that users can list dinosaurs instances created by a member of their org
	account = h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// get populated list of dinosaur requests

	afterPostList, _, err = client.DefaultApi.GetCentrals(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(afterPostList.Items)).To(Equal(1), "Expected dinosaur requests list length to be 1")
	Expect(afterPostList.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(afterPostList.Total).To(Equal(int32(1)), "Expected Total == 1")

	// get dinosaur request item from the list
	listItem = afterPostList.Items[0]

	// check whether the seedDinosaur properties are the same as those from the dinosaur request list item
	Expect(seedDinosaur.Id).To(Equal(listItem.Id))
	Expect(seedDinosaur.Kind).To(Equal(listItem.Kind))
	Expect(listItem.Kind).To(Equal(presenters.KindDinosaur))
	Expect(seedDinosaur.Href).To(Equal(listItem.Href))
	Expect(seedDinosaur.Region).To(Equal(listItem.Region))
	Expect(listItem.Region).To(Equal(mocks.MockCluster.Region().ID()))
	Expect(seedDinosaur.CloudProvider).To(Equal(listItem.CloudProvider))
	Expect(listItem.CloudProvider).To(Equal(mocks.MockCluster.CloudProvider().ID()))
	Expect(seedDinosaur.Name).To(Equal(listItem.Name))
	Expect(listItem.Name).To(Equal(mockDinosaurName))
	Expect(listItem.Status).To(Equal(constants2.DinosaurRequestStatusAccepted.String()))

	// new account setup to prove that users can only list their own (the one they created and the one created by a member of their org) dinosaur instances
	// this value is taken from config/quota-management-list-configuration.yaml
	anotherOrgID := "13639843"
	account = h.NewAccount(h.NewID(), faker.Name(), faker.Email(), anotherOrgID)
	ctx = h.NewAuthenticatedContext(account, nil)

	// expecting empty list for user that hasn't created any dinosaurs yet
	newUserList, _, err := client.DefaultApi.GetCentrals(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list dinosaur requests:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(newUserList.Items)).To(Equal(0), "Expected dinosaur requests list length to be 0")
	Expect(newUserList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(newUserList.Total).To(Equal(int32(0)), "Expected Total == 0")
}
*/