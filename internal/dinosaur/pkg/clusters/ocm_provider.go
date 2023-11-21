package clusters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

const (
	ipdAlreadyCreatedErrorToCheck = "already exists"
)

// OCMProvider ...
type OCMProvider struct {
	ocmClient      ocm.Client
	clusterBuilder ClusterBuilder
	ocmConfig      *ocm.OCMConfig
}

// blank assignment to verify that OCMProvider implements Provider
var _ Provider = &OCMProvider{}

// Create ...
func (o *OCMProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	// Build a new OSD cluster object
	newCluster, err := o.clusterBuilder.NewOCMClusterFromCluster(request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build OCM cluster")
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := o.ocmClient.CreateCluster(newCluster)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create OCM cluster")
	}

	result := &types.ClusterSpec{
		Status: api.ClusterProvisioning,
	}
	if createdCluster.ID() != "" {
		result.InternalID = createdCluster.ID()
	}
	if createdCluster.ExternalID() != "" {
		result.ExternalID = createdCluster.ExternalID()
	}
	return result, nil
}

// CheckClusterStatus ...
func (o *OCMProvider) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	ocmCluster, err := o.ocmClient.GetCluster(spec.InternalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster %s", spec.InternalID)
	}
	clusterStatus := ocmCluster.Status()
	if spec.Status == "" {
		spec.Status = api.ClusterProvisioning
	}

	spec.StatusDetails = clusterStatus.ProvisionErrorMessage()

	if clusterStatus.State() == clustersmgmtv1.ClusterStateReady {
		if spec.ExternalID == "" {
			externalID, ok := ocmCluster.GetExternalID()
			if !ok {
				return nil, errors.Errorf("External ID for cluster %s cannot be found", ocmCluster.ID())
			}
			spec.ExternalID = externalID
		}
		spec.Status = api.ClusterProvisioned
	}
	if clusterStatus.State() == clustersmgmtv1.ClusterStateError {
		spec.Status = api.ClusterFailed
	}
	return spec, nil
}

// Delete ...
func (o *OCMProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	code, err := o.ocmClient.DeleteCluster(spec.InternalID)
	if err != nil && code != http.StatusNotFound {
		return false, errors.Wrapf(err, "failed to delete cluster %s", spec.InternalID)
	}
	return code == http.StatusNotFound, nil
}

// GetClusterDNS ...
func (o *OCMProvider) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	clusterDNS, err := o.ocmClient.GetClusterDNS(clusterSpec.InternalID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get dns for cluster %s", clusterSpec.InternalID)
	}
	return clusterDNS, nil
}

// InstallDinosaurOperator ...
func (o *OCMProvider) InstallDinosaurOperator(clusterSpec *types.ClusterSpec) (bool, error) {
	return o.installAddon(clusterSpec, o.ocmConfig.CentralOperatorAddonID)
}

// InstallFleetshard ...
func (o *OCMProvider) InstallFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	return o.installAddonWithParams(clusterSpec, o.ocmConfig.FleetshardAddonID, params)
}

func (o *OCMProvider) installAddon(clusterSpec *types.ClusterSpec, addonID string) (bool, error) {
	clusterID := clusterSpec.InternalID
	addonInstallation, err := o.ocmClient.GetAddon(clusterID, addonID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get addon %s for cluster %s", addonID, clusterSpec.InternalID)
	}

	// Addon needs to be installed if addonInstallation doesn't exist
	if addonInstallation.ID() == "" {
		addonInstallation, err = o.ocmClient.CreateAddon(clusterID, addonID)
		if err != nil {
			return false, errors.Wrapf(err, "failed to create addon %s for cluster %s", addonID, clusterSpec.InternalID)
		}
	}

	// The cluster is ready when the state reports ready
	if addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		return true, nil
	}

	return false, nil
}

func (o *OCMProvider) installAddonWithParams(clusterSpec *types.ClusterSpec, addonID string, params []types.Parameter) (bool, error) {
	addonInstallation, addonErr := o.ocmClient.GetAddon(clusterSpec.InternalID, addonID)
	if addonErr != nil {
		return false, errors.Wrapf(addonErr, "failed to get addon %s for cluster %s", addonID, clusterSpec.InternalID)
	}

	if addonInstallation != nil && addonInstallation.ID() == "" {
		glog.V(5).Infof("No existing %s addon found, create a new one", addonID)
		addonInstallation, addonErr = o.ocmClient.CreateAddonWithParams(clusterSpec.InternalID, addonID, params)
		if addonErr != nil {
			return false, errors.Wrapf(addonErr, "failed to create addon %s for cluster %s", addonID, clusterSpec.InternalID)
		}
	}

	if addonInstallation != nil && addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		addonInstallation, addonErr = o.ocmClient.UpdateAddonParameters(clusterSpec.InternalID, addonInstallation.ID(), params)
		if addonErr != nil {
			return false, errors.Wrapf(addonErr, "failed to update parameters for addon %s on cluster %s", addonInstallation.ID(), clusterSpec.InternalID)
		}
		return true, nil
	}

	return false, nil
}

// GetCloudProviders ...
func (o *OCMProvider) GetCloudProviders() (*types.CloudProviderInfoList, error) {
	list := types.CloudProviderInfoList{}
	providerList, err := o.ocmClient.GetCloudProviders()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cloud providers from OCM")
	}
	var items []types.CloudProviderInfo
	providerList.Each(func(item *clustersmgmtv1.CloudProvider) bool {
		p := types.CloudProviderInfo{
			ID:          item.ID(),
			Name:        item.Name(),
			DisplayName: item.DisplayName(),
		}
		items = append(items, p)
		return true
	})
	list.Items = items
	return &list, nil
}

// GetCloudProviderRegions ...
func (o *OCMProvider) GetCloudProviderRegions(providerInfo types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
	list := types.CloudProviderRegionInfoList{}
	cp, err := clustersmgmtv1.NewCloudProvider().ID(providerInfo.ID).Name(providerInfo.Name).DisplayName(providerInfo.DisplayName).Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build cloud provider object")
	}
	regionsList, err := o.ocmClient.GetRegions(cp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get regions for provider %s", providerInfo.Name)
	}
	var items []types.CloudProviderRegionInfo
	regionsList.Each(func(item *clustersmgmtv1.CloudRegion) bool {
		r := types.CloudProviderRegionInfo{
			ID:              item.ID(),
			CloudProviderID: item.CloudProvider().ID(),
			Name:            item.Name(),
			DisplayName:     item.DisplayName(),
			SupportsMultiAZ: item.SupportsMultiAZ(),
		}
		items = append(items, r)
		return true
	})
	list.Items = items
	return &list, nil
}

// ensure OCMProvider implements Provider interface
var _ Provider = &OCMProvider{}

func newOCMProvider(ocmClient ocm.ClusterManagementClient, clusterBuilder ClusterBuilder, ocmConfig *ocm.OCMConfig) *OCMProvider {
	return &OCMProvider{
		ocmClient:      ocmClient,
		clusterBuilder: clusterBuilder,
		ocmConfig:      ocmConfig,
	}
}
