package integration

import (
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

/* TODO(ROX-10709)
import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/api"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test/common"

	coreTest "github.com/stackrox/acs-fleet-manager/test"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	mockDinosaurName1 = "test-dinosaur1"
	mockDinosaurName2 = "a-dinosaur1"
	mockDinosaurName3 = "z-dinosaur1"
	mockDinosaurName4 = "b-dinosaur1"
)

type TestServer struct {
	OcmServer     *httptest.Server
	TearDown      func()
	ClusterID     string
	Token         string
	Client        *public.APIClient
	PrivateClient *private.APIClient
	Helper        *coreTest.Helper
	Ctx           context.Context
}

type claimsFunc func(account *v1.Account, clusterId string, h *coreTest.Helper) jwt.MapClaims

var clusterId = api.NewID()

func setup(t *testing.T, claims claimsFunc, startupHook interface{}) TestServer {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, startupHook)
	db := test.TestServices.DBFactory.New()
	// create a dummy cluster that will be used throughout the test
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterId,
		},
		ClusterID:             clusterId,
		MultiAZ:               true,
		Region:                "baremetal",
		CloudProvider:         "baremetal",
		Status:                api.ClusterReady,
		IdentityProviderID:    "some-id",
		ClusterDNS:            "some-cluster.dns.org",
		ProviderType:          api.ClusterProviderStandalone,
		SupportedInstanceType: api.AllInstanceTypeSupport.String(),
	}

	if err := db.Create(cluster).Error; err != nil {
		t.Fatalf("failed to create dummy cluster")
	}

	account := h.NewAllowedServiceAccount()
	ctx := h.NewAuthenticatedContext(account, claims(account, clusterId, h))
	token := h.CreateJWTStringWithClaim(account, claims(account, clusterId, h))

	config := private.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", test.TestServices.ServerConfig.BindAddress)
	config.DefaultHeader = map[string]string{
		"Authorization": "Bearer " + token,
	}
	privateClient := private.NewAPIClient(config)

	return TestServer{
		OcmServer:     ocmServer,
		Client:        client,
		PrivateClient: privateClient,
		TearDown: func() {
			ocmServer.Close()
			tearDown()
		},
		ClusterID: clusterId,
		Token:     token,
		Helper:    h,
		Ctx:       ctx,
	}
}

func TestDataPlaneEndpoints_GetAndUpdateManagedDinosaurs(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"fleetshard_operator"},
			},
			"fleetshard-operator-cluster-id": cid,
		}
	}, nil)
	defer testServer.TearDown()
	dinosaurHost := "some-dinosaur-host"

	var testDinosaurs = []*dbapi.DinosaurRequest{
		{
			ClusterID:                      testServer.ClusterID,
			MultiAZ:                        false,
			Name:                           mockDinosaurName1,
			Namespace:                      "mk-1",
			Status:                         constants2.DinosaurRequestStatusDeprovision.String(),
			Host:                           dinosaurHost,
			DesiredDinosaurVersion:         "2.7.0",
			DesiredDinosaurOperatorVersion: "dinosaur-operator.v0.23.0-0",
			InstanceType:                   types.STANDARD.String(),
		},
		{
			ClusterID:                      testServer.ClusterID,
			MultiAZ:                        false,
			Name:                           mockDinosaurName2,
			Namespace:                      "mk-2",
			Status:                         constants2.DinosaurRequestStatusProvisioning.String(),
			Host:                           dinosaurHost,
			DesiredDinosaurVersion:         "2.6.0",
			DesiredDinosaurOperatorVersion: "dinosaur-operator.v0.23.0-0",
			InstanceType:                   types.STANDARD.String(),
		},
		{
			ClusterID:                      testServer.ClusterID,
			MultiAZ:                        false,
			Name:                           mockDinosaurName3,
			Namespace:                      "mk-3",
			Status:                         constants2.DinosaurRequestStatusPreparing.String(),
			Host:                           dinosaurHost,
			DesiredDinosaurVersion:         "2.7.1",
			DesiredDinosaurOperatorVersion: "dinosaur-operator.v0.23.0-0",
			InstanceType:                   types.EVAL.String(),
		},
		{
			ClusterID:                      testServer.ClusterID,
			MultiAZ:                        false,
			Name:                           mockDinosaurName4,
			Namespace:                      "mk-4",
			Status:                         constants2.DinosaurRequestStatusReady.String(),
			Host:                           dinosaurHost,
			DesiredDinosaurVersion:         "2.7.2",
			DesiredDinosaurOperatorVersion: "dinosaur-operator.v0.23.0-0",
			InstanceType:                   types.STANDARD.String(),
		},
		{
			ClusterID:                      testServer.ClusterID,
			MultiAZ:                        false,
			Namespace:                      "mk-5",
			Name:                           mockDinosaurName4,
			Status:                         constants2.DinosaurRequestStatusFailed.String(),
			Host:                           dinosaurHost,
			DesiredDinosaurVersion:         "2.7.2",
			DesiredDinosaurOperatorVersion: "dinosaur-operator.v0.23.0-0",
			InstanceType:                   types.STANDARD.String(),
		},
	}

	db := test.TestServices.DBFactory.New()

	// create dummy dinosaurs
	if err := db.Create(&testDinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// create an additional dinosaur in failed state without host. This indicates that the
	// dinosaur failed in preparing state and should not be returned in the list
	additionalDinosaur := &dbapi.DinosaurRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockDinosaurName4,
		Namespace:              "mk",
		Status:                 constants2.DinosaurRequestStatusFailed.String(),
		DesiredDinosaurVersion: "2.7.2",
		InstanceType:           types.EVAL.String(),
	}

	if err := db.Save(additionalDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetDinosaurs(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(4)) // only count valid Managed Dinosaur CR

	for _, k := range testDinosaurs {
		if k.Status != constants2.DinosaurRequestStatusPreparing.String() {
			if mk := findManagedDinosaurByID(list.Items, k.ID); mk != nil {
				Expect(mk.Metadata.Name).To(Equal(k.Name))
				Expect(mk.Metadata.Annotations.MasPlacementId).To(Equal(k.PlacementId))
				Expect(mk.Metadata.Annotations.MasId).To(Equal(k.ID))
				Expect(mk.Metadata.Namespace).NotTo(BeEmpty())
				Expect(mk.Spec.Deleted).To(Equal(k.Status == constants2.DinosaurRequestStatusDeprovision.String()))
				Expect(mk.Spec.Versions.Dinosaur).To(Equal(k.DesiredDinosaurVersion))
				Expect(mk.Spec.Endpoint.Tls).To(BeNil())
			} else {
				t.Error("failed matching manageddinosaur id with dinosaurrequest id")
				break
			}
		}
	}

	var readyClusters, deletedClusters []string
	updates := map[string]private.DataPlaneDinosaurStatus{}
	condtionsReasons := []string{
		"DinosaurOperatorUpdating",
		"DinosaurUpdating",
	}
	lengthConditionsReasons := len(condtionsReasons)
	for idx, item := range list.Items {
		if !item.Spec.Deleted {
			updates[item.Metadata.Annotations.MasId] = private.DataPlaneDinosaurStatus{
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "True",
					Reason: condtionsReasons[idx%lengthConditionsReasons],
				}},
				Versions: private.DataPlaneDinosaurStatusVersions{
					Dinosaur:         fmt.Sprintf("dinosaur-new-version-%s", item.Metadata.Annotations.MasId),
					DinosaurOperator: fmt.Sprintf("dinosaur-operator-new-version-%s", item.Metadata.Annotations.MasId),
				},
			}
			readyClusters = append(readyClusters, item.Metadata.Annotations.MasId)
		} else {
			updates[item.Metadata.Annotations.MasId] = private.DataPlaneDinosaurStatus{
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "False",
					Reason: "Deleted",
				}},
			}
			deletedClusters = append(deletedClusters, item.Metadata.Annotations.MasId)
		}
	}

	// routes will be stored the first time status are updated
	_, err = testServer.PrivateClient.AgentClustersApi.UpdateDinosaurClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	Expect(err).NotTo(HaveOccurred())

	// wait for the CNAMEs for routes to be created
	waitErr := common.NewPollerBuilder(test.TestServices.DBFactory).
		IntervalAndTimeout(1*time.Second, 1*time.Minute).
		RetryLogMessage("waiting for Dinosaur routes to be created").
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			c := &dbapi.DinosaurRequest{}
			if err := db.First(c, "routes IS NOT NULL").Error; err != nil {
				return false, err
			}
			// if one route is created, it is safe to assume all routes are created
			return c.RoutesCreated, nil
		}).Build().Poll()
	Expect(waitErr).To(BeNil())

	// Send the requests again, this time the instances should be ready because routes are created
	_, err = testServer.PrivateClient.AgentClustersApi.UpdateDinosaurClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	Expect(err).NotTo(HaveOccurred())

	for _, cid := range readyClusters {
		c := &dbapi.DinosaurRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}

		sentUpdate, ok := updates[cid]
		if !ok {
			t.Errorf("failed to find sent dinosaur status update related to cluster with id %s", cid)
		}

		var sentReadyCondition string
		for _, cond := range sentUpdate.Conditions {
			if cond.Type == "Ready" {
				sentReadyCondition = cond.Reason
			}
		}
		Expect(sentReadyCondition).NotTo(BeEmpty())

		// Test version related reported fields
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))
		Expect(c.ActualDinosaurVersion).To(Equal(sentUpdate.Versions.Dinosaur))
		Expect(c.ActualDinosaurOperatorVersion).To(Equal(sentUpdate.Versions.DinosaurOperator))
		Expect(c.DinosaurOperatorUpgrading).To(Equal(sentReadyCondition == "DinosaurOperatorUpdating"))
		Expect(c.DinosaurUpgrading).To(Equal(sentReadyCondition == "DinosaurUpdating"))

		// TODO test when dinosaur is being upgraded when fleet shard operator side
		// appropriately reports it
	}

	for _, cid := range deletedClusters {
		c := &dbapi.DinosaurRequest{}
		// need to use Unscoped here as there is a chance the entry is soft deleted already
		if err := db.Unscoped().Where("id = ?", cid).First(c).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusDeleting.String()))
	}

	for _, cid := range readyClusters {
		// update the status to ready again and remove reason field to simulate the end of upgrade process as reported by fleet-shard
		_, err = testServer.PrivateClient.AgentClustersApi.UpdateDinosaurClusterStatus(testServer.Ctx, testServer.ClusterID, map[string]private.DataPlaneDinosaurStatus{
			cid: {
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "True",
				}},
				Versions: private.DataPlaneDinosaurStatusVersions{
					Dinosaur:         fmt.Sprintf("dinosaur-new-version-%s", cid),
					DinosaurOperator: fmt.Sprintf("dinosaur-operator-new-version-%s", cid),
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())

		c := &dbapi.DinosaurRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}

		// Make sure that the dinosaur stays in ready state and status of dinosaur operator upgrade is false.
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))
		Expect(c.DinosaurOperatorUpgrading).To(BeFalse())
	}
}

func findManagedDinosaurByID(slice []private.ManagedDinosaur, dinosaurId string) *private.ManagedDinosaur {
	match := func(item private.ManagedDinosaur) bool { return item.Metadata.Annotations.MasId == dinosaurId }
	for _, item := range slice {
		if match(item) {
			return &item
		}
	}
	return nil
}
*/

func mockedClusterWithMetricsInfo(computeNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterNodeBuilder := clustersmgmtv1.NewClusterNodes()
	clusterNodeBuilder.Compute(computeNodes)
	clusterBuilder.Nodes(clusterNodeBuilder)
	return clusterBuilder.Build()
}
