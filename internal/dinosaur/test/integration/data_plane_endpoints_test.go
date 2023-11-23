package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

const (
	issuerURL = "https://sso.redhat.com/auth/realms/redhat-external"
	orgID     = "11009103"
)

func mockedClusterWithMetricsInfo(computeNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := mocks.GetMockClusterBuilder(nil)
	clusterNodeBuilder := clustersmgmtv1.NewClusterNodes()
	clusterNodeBuilder.Compute(computeNodes)
	clusterBuilder.Nodes(clusterNodeBuilder)
	return clusterBuilder.Build()
}

func TestDataPlaneClusterStatus(t *testing.T) {
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

	clusterID := api.NewID()
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss":    issuerURL,
		"org_id": orgID,
	}
	ctx := h.NewAuthenticatedContext(account, claims)
	token := h.CreateJWTStringWithClaim(account, claims)

	config := private.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", test.TestServices.ServerConfig.BindAddress)
	config.DefaultHeader = map[string]string{
		"Authorization": "Bearer " + token,
	}
	privateClient := private.NewAPIClient(config)

	db := test.TestServices.DBFactory.New()
	cluster := &api.Cluster{
		Meta: api.Meta{
			ID: clusterID,
		},
		ClusterID:          clusterID,
		MultiAZ:            true,
		Region:             "baremetal",
		CloudProvider:      "baremetal",
		Status:             api.ClusterReady,
		IdentityProviderID: "some-id",
		ClusterDNS:         "some-cluster-dns",
		ProviderType:       api.ClusterProviderStandalone,
	}

	err = db.Create(cluster).Error
	Expect(err).NotTo(HaveOccurred())

	resp, err := privateClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, clusterID, private.DataPlaneClusterUpdateStatusRequest{
		FleetshardAddonStatus: private.DataPlaneClusterUpdateStatusRequestFleetshardAddonStatus{
			Version:             "0.2.0",
			SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
			PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
			ParametersSHA256Sum: "f54d2c5cb370f4f87a31ccd8f72d97a85d89838720bd69278d1d40ee1cea00dc", // pragma: allowlist secret
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	clusterDetails := &api.Cluster{
		ClusterID: cluster.ID,
	}

	err = db.Where(clusterDetails).First(cluster).Error
	Expect(err).ToNot(HaveOccurred())

	Expect(cluster.FleetshardAddonStatus).NotTo(BeEmpty())

	var fleetshardAddonStatus dbapi.FleetshardAddonStatus
	err = json.Unmarshal(cluster.FleetshardAddonStatus, &fleetshardAddonStatus)
	Expect(err).ToNot(HaveOccurred())

	Expect(fleetshardAddonStatus.Version).To(Equal("0.2.0"))
	Expect(fleetshardAddonStatus.SourceImage).To(Equal("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac"))
	Expect(fleetshardAddonStatus.PackageImage).To(Equal("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c"))
	Expect(fleetshardAddonStatus.ParametersSHA256Sum).To(Equal("f54d2c5cb370f4f87a31ccd8f72d97a85d89838720bd69278d1d40ee1cea00dc")) // pragma: allowlist secret
}
