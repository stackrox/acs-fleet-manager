package integration

import (
	"net/http/httptest"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"

	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const gcp = "gcp"
const aws = "aws"
const azure = "azure"

const afEast1Region = "af-east-1"
const usEast1Region = "us-east-1"

var limit = int(5)

var allTypesMap = config.InstanceTypeMap{
	"eval": {
		Limit: &limit,
	},
	"standard": {
		Limit: &limit,
	},
}

var standardMap = config.InstanceTypeMap{
	"standard": {
		Limit: &limit,
	},
}

var evalMap = config.InstanceTypeMap{
	"eval": {
		Limit: &limit,
	},
}

var noneTypeMap = config.InstanceTypeMap{}

var dummyClusters = []*api.Cluster{
	{
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             afEast1Region,
		CloudProvider:      gcp,
		Status:             api.ClusterReady,
		ProviderType:       api.ClusterProviderStandalone,
		IdentityProviderID: "some-identity-provider-id",
	},
	{
		ClusterID:          api.NewID(),
		MultiAZ:            true,
		Region:             usEast1Region,
		CloudProvider:      aws,
		Status:             api.ClusterReady,
		ProviderType:       api.ClusterProviderOCM,
		IdentityProviderID: "some-identity-provider-id",
	},
}

func setupOcmServerWithMockRegionsResp() (*httptest.Server, error) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	usEast1 := clustersmgmtv1.NewCloudRegion().
		ID("us-east-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1").
		DisplayName("us east 1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	afSouth1 := clustersmgmtv1.NewCloudRegion().
		ID("af-south-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/af-south-1").
		DisplayName("af-south-1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	euWest2 := clustersmgmtv1.NewCloudRegion().
		ID("eu-west-2").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/eu-west-2").
		DisplayName("eu-west-2").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	euCentral1 := clustersmgmtv1.NewCloudRegion().
		ID("eu-central-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/eu-central-1").
		DisplayName("eu-central-1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	apSouth1 := clustersmgmtv1.NewCloudRegion().
		ID("ap-south-1").
		HREF("/api/clusters_mgmt/v1/cloud_providers/aws/regions/ap-south-1").
		DisplayName("ap-south-1").
		CloudProvider(mocks.GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	awsRegions, err := clustersmgmtv1.NewCloudRegionList().Items(usEast1, afSouth1, euWest2, euCentral1, apSouth1).Build()
	if err != nil {
		return nil, err
	}
	ocmServerBuilder.SetCloudRegionsGetResponse(awsRegions, nil)
	ocmServer := ocmServerBuilder.Build()
	return ocmServer, nil
}

func TestCloudProviderRegions(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	cloudProviderRegions, err := test.TestServices.CloudProvidersService.GetCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == aws || regions.ID == azure || regions.ID == gcp {
			Expect(len(regions.RegionList.Items)).NotTo(Equal(0))
		}
		for _, r := range regions.RegionList.Items {
			id := r.ID
			name := r.DisplayName
			multiAz := r.SupportsMultiAZ

			Expect(regions.ID).NotTo(Equal(nil))
			Expect(id).NotTo(Equal(nil))
			Expect(name).NotTo(Equal(nil))
			Expect(multiAz).NotTo(Equal(nil))
		}
	}

}

func TestCachedCloudProviderRegions(t *testing.T) {
	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	_, _, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	cloudProviderRegions, err := test.TestServices.CloudProvidersService.GetCachedCloudProvidersWithRegions()
	Expect(err).NotTo(HaveOccurred(), "Error:  %v", err)

	for _, regions := range cloudProviderRegions {
		// regions.ID == "baremetal" | "libvirt" | "openstack" | "vsphere" have empty region list
		if regions.ID == aws || regions.ID == azure || regions.ID == gcp {
			Expect(len(regions.RegionList.Items)).NotTo(Equal(0))
		}
		for _, r := range regions.RegionList.Items {
			id := r.ID
			name := r.DisplayName
			multiAz := r.SupportsMultiAZ

			Expect(regions.ID).NotTo(Equal(nil))
			Expect(id).NotTo(Equal(nil))
			Expect(name).NotTo(Equal(nil))
			Expect(multiAz).NotTo(Equal(nil))
		}
	}

}
