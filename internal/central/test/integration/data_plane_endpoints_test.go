package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/test"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
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
		t.Fatalf("%s", err.Error())
	}

	ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()
	h, _, tearDown := test.NewCentralHelper(t, ocmServer)
	defer tearDown()

	clusterID := api.NewID()
	token := h.CreateDataPlaneJWTString()

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

	resp, err := privateClient.AgentClustersApi.UpdateAgentClusterStatus(context.Background(), clusterID, private.DataPlaneClusterUpdateStatusRequest{
		Addons: []private.DataPlaneClusterUpdateStatusRequestAddons{
			{
				Id:                  "acs-fleetshard",
				Version:             "0.2.0",
				SourceImage:         "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
				PackageImage:        "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
				ParametersSHA256Sum: "f54d2c5cb370f4f87a31ccd8f72d97a85d89838720bd69278d1d40ee1cea00dc", // pragma: allowlist secret
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	clusterDetails := &api.Cluster{
		ClusterID: cluster.ID,
	}

	err = db.Where(clusterDetails).First(cluster).Error
	Expect(err).ToNot(HaveOccurred())

	Expect(cluster.Addons).NotTo(BeEmpty())

	var addonInstallations []dbapi.AddonInstallation
	err = json.Unmarshal(cluster.Addons, &addonInstallations)
	Expect(err).ToNot(HaveOccurred())
	Expect(addonInstallations).To(HaveLen(1))

	fleetshardAddon := addonInstallations[0]
	Expect(fleetshardAddon.ID).To(Equal("acs-fleetshard"))
	Expect(fleetshardAddon.Version).To(Equal("0.2.0"))
	Expect(fleetshardAddon.SourceImage).To(Equal("quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac"))
	Expect(fleetshardAddon.PackageImage).To(Equal("quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c"))
	Expect(fleetshardAddon.ParametersSHA256Sum).To(Equal("f54d2c5cb370f4f87a31ccd8f72d97a85d89838720bd69278d1d40ee1cea00dc")) // pragma: allowlist secret
}
