package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antihax/optional"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
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

func TestListCloudProviders(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	cloudProviderList, resp, err := client.DefaultApi.GetCloudProviders(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderList.Items).NotTo(BeEmpty(), "Expected cloud providers list")

	// verify that the cloud providers list should contain atleast "gcp" which comes from standalone provider type
	hasGcp := false
	for _, cloudProvider := range cloudProviderList.Items {
		if cloudProvider.Id == gcp {
			hasGcp = true
			break
		}
	}
	Expect(hasGcp).To(BeTrue())
}

func TestListCloudProviderRegions(t *testing.T) {
	ocmServer, err := setupOcmServerWithMockRegionsResp()
	if err != nil {
		t.Errorf("Failed to set mock ocm region list response")
	}
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(pc *config.ProviderConfig) {
		pc.ProvidersConfig.SupportedProviders = config.ProviderList{
			{
				Name:    "aws",
				Default: true,
				Regions: config.RegionList{
					{
						Name:                   "us-east-1",
						Default:                true,
						SupportedInstanceTypes: allTypesMap,
					},
					{
						Name:                   "af-south-1",
						SupportedInstanceTypes: standardMap,
					},
					{
						Name:                   "eu-central-1",
						SupportedInstanceTypes: noneTypeMap,
					},
				},
			},
		}
	})
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// all regions available for the 'aws' cloud provider should be returned
	cloudProviderRegionsList, resp1, err := client.DefaultApi.GetCloudProviderRegions(ctx, mocks.MockCluster.CloudProvider().ID(), nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", err)
	Expect(resp1.StatusCode).To(Equal(http.StatusOK))
	Expect(cloudProviderRegionsList.Items).NotTo(BeEmpty(), "Expected aws cloud provider regions to return a non-empty list")

	// enabled should only be set to true for regions that support at least one instance type in the providers config
	for _, cpr := range cloudProviderRegionsList.Items {
		if cpr.Id == "us-east-1" || cpr.Id == "af-south-1" {
			Expect(cpr.Enabled).To(BeTrue())
		} else {
			Expect(cpr.Enabled).To(BeFalse())
		}
	}

	// all regions available for the 'gcp' cloud provider should be returned
	gcpCloudProviderRegions, gcpResp, gcpErr := client.DefaultApi.GetCloudProviderRegions(ctx, gcp, nil)
	Expect(gcpErr).NotTo(HaveOccurred(), "Error occurred when attempting to list gcp cloud providers regions:  %v", gcpErr)
	Expect(gcpResp.StatusCode).To(Equal(http.StatusOK))
	Expect(gcpCloudProviderRegions.Items).NotTo(BeEmpty(), "Expected gcp cloud provider regions to return a non-empty list")

	// all gcp regions returned should have enabled set to false as they do not support any instance types as specified in the providers config
	for _, cpr := range gcpCloudProviderRegions.Items {
		Expect(cpr.Enabled).To(BeFalse())
	}

	// test with wrong provider id
	wrongCloudProviderList, respFromWrongID, errFromWrongID := client.DefaultApi.GetCloudProviderRegions(ctx, "wrong_provider_id", nil)
	Expect(errFromWrongID).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud providers regions:  %v", errFromWrongID)
	Expect(respFromWrongID.StatusCode).To(Equal(http.StatusOK))
	Expect(wrongCloudProviderList.Items).To(BeEmpty(), "Expected cloud providers regions list empty")
}

func TestListCloudProviderRegionsWithInstanceType(t *testing.T) {
	ocmServer, err := setupOcmServerWithMockRegionsResp()
	if err != nil {
		t.Errorf("Failed to set mock ocm region list response")
	}
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(pc *config.ProviderConfig) {
		pc.ProvidersConfig.SupportedProviders = config.ProviderList{
			{
				Name:    "aws",
				Default: true,
				Regions: config.RegionList{
					{
						Name:                   "us-east-1",
						Default:                true,
						SupportedInstanceTypes: allTypesMap,
					},
					{
						Name:                   "af-south-1",
						SupportedInstanceTypes: standardMap,
					},
					{
						Name:                   "eu-west-2",
						SupportedInstanceTypes: evalMap,
					},
					{
						Name:                   "eu-central-1",
						SupportedInstanceTypes: noneTypeMap,
					},
				},
			},
		}
	})
	defer teardown()

	// Create two clusters each with different provider type
	if err := test.TestServices.DBFactory.New().Create(dummyClusters).Error; err != nil {
		t.Error("failed to create dummy clusters")
		return
	}

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	// should only return regions that support 'eval' instance type if 'instance_type=eval' was specified
	regions, resp, err := client.DefaultApi.GetCloudProviderRegions(ctx, "aws", &public.GetCloudProviderRegionsOpts{
		InstanceType: optional.NewString("eval"),
	})
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud provider regions of instance type 'eval': %v", err)
	Expect(regions.Items).To(HaveLen(2))
	for _, r := range regions.Items {
		Expect(r.Id).To(SatisfyAny(Equal("us-east-1"), Equal("eu-west-2")))
		Expect(r.Enabled).To(Equal(true))
	}

	// should only return regions that support 'standard' instance types if 'instance_type=standard' was specified
	regions, resp, err = client.DefaultApi.GetCloudProviderRegions(ctx, "aws", &public.GetCloudProviderRegionsOpts{
		InstanceType: optional.NewString("standard"),
	})
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud provider regions of instance type 'standard': %v", err)
	Expect(regions.Items).To(HaveLen(2))
	for _, r := range regions.Items {
		Expect(r.Id).To(SatisfyAny(Equal("us-east-1"), Equal("af-south-1")))
		Expect(r.Enabled).To(Equal(true))
	}

	// should not return any regions if specified instance_type was not valid
	regions, resp, err = client.DefaultApi.GetCloudProviderRegions(ctx, "aws", &public.GetCloudProviderRegionsOpts{
		InstanceType: optional.NewString("!invalid!"),
	})
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud provider regions of instance type '!invalid!': %v", err)
	Expect(regions.Items).To(HaveLen(0))

	// should return all regions if specified instance_type was an empty string
	regions, resp, err = client.DefaultApi.GetCloudProviderRegions(ctx, "aws", &public.GetCloudProviderRegionsOpts{
		InstanceType: optional.NewString(""),
	})
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list cloud provider regions of instance type '!invalid!': %v", err)
	Expect(regions.Items).ToNot(BeEmpty())
	for _, cpr := range regions.Items {
		if cpr.Id == "us-east-1" || cpr.Id == "af-south-1" || cpr.Id == "eu-west-2" {
			Expect(cpr.Enabled).To(BeTrue())
		} else {
			Expect(cpr.Enabled).To(BeFalse())
		}
	}
}
