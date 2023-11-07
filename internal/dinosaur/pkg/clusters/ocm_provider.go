package clusters

import (
	"fmt"
	"net/http"
	"strings"

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

// AddIdentityProvider ...
func (o *OCMProvider) AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProviderInfo types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
	if identityProviderInfo.OpenID != nil {
		idpID, err := o.addOpenIDIdentityProvider(clusterSpec, *identityProviderInfo.OpenID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add identity provider for cluster %s", clusterSpec.InternalID)
		}
		identityProviderInfo.OpenID.ID = idpID
		return &identityProviderInfo, nil
	}
	return nil, nil
}

// ScaleUp ...
func (o *OCMProvider) ScaleUp(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error) {
	_, err := o.ocmClient.ScaleUpComputeNodes(clusterSpec.InternalID, increment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scale up cluster %s by %d nodes", clusterSpec.InternalID, increment)
	}
	return clusterSpec, nil
}

// ScaleDown ...
func (o *OCMProvider) ScaleDown(clusterSpec *types.ClusterSpec, decrement int) (*types.ClusterSpec, error) {
	_, err := o.ocmClient.ScaleDownComputeNodes(clusterSpec.InternalID, decrement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to scale down cluster %s by %d nodes", clusterSpec.InternalID, decrement)
	}
	return clusterSpec, nil
}

// SetComputeNodes ...
func (o *OCMProvider) SetComputeNodes(clusterSpec *types.ClusterSpec, numNodes int) (*types.ClusterSpec, error) {
	_, err := o.ocmClient.SetComputeNodes(clusterSpec.InternalID, numNodes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to set compute nodes value of cluster %s to %d", clusterSpec.InternalID, numNodes)
	}
	return clusterSpec, nil
}

// GetComputeNodes ...
func (o *OCMProvider) GetComputeNodes(clusterSpec *types.ClusterSpec) (*types.ComputeNodesInfo, error) {
	ocmCluster, err := o.ocmClient.GetCluster(clusterSpec.InternalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster details %s", clusterSpec.InternalID)
	}
	metrics, err := o.ocmClient.GetExistingClusterMetrics(clusterSpec.InternalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get metrics for cluster %s", clusterSpec.InternalID)
	}
	if metrics == nil {
		return nil, errors.Errorf("cluster ID %s has no metrics", clusterSpec.InternalID)
	}
	existingNodes, ok := metrics.GetNodes()
	if !ok {
		return nil, errors.Errorf("Cluster ID %s has no node metrics", clusterSpec.InternalID)
	}

	existingComputeNodes, ok := existingNodes.GetCompute()
	if !ok {
		return nil, errors.Errorf("Cluster ID %s has no compute node metrics", clusterSpec.InternalID)
	}

	desiredNodes, ok := ocmCluster.GetNodes()
	if !ok {
		return nil, errors.Errorf("Cluster ID %s has no desired node information", clusterSpec.InternalID)
	}
	desiredComputeNodes, ok := desiredNodes.GetCompute()
	if !ok {
		return nil, errors.Errorf("Cluster ID %s has no desired compute node information", clusterSpec.InternalID)
	}
	return &types.ComputeNodesInfo{
		Actual:  int(existingComputeNodes),
		Desired: desiredComputeNodes,
	}, nil
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

func (o *OCMProvider) addOpenIDIdentityProvider(clusterSpec *types.ClusterSpec, openIDIdpInfo types.OpenIDIdentityProviderInfo) (string, error) {
	provider, buildErr := buildIdentityProvider(openIDIdpInfo)
	if buildErr != nil {
		return "", errors.WithStack(buildErr)
	}
	createdIdentityProvider, createIdentityProviderErr := o.ocmClient.CreateIdentityProvider(clusterSpec.InternalID, provider)
	if createIdentityProviderErr != nil {
		// check to see if identity provider with name 'Dinosaur_SRE' already exists, if so use it.
		if strings.Contains(createIdentityProviderErr.Error(), ipdAlreadyCreatedErrorToCheck) {
			identityProvidersList, identityProviderListErr := o.ocmClient.GetIdentityProviderList(clusterSpec.InternalID)
			if identityProviderListErr != nil {
				return "", errors.WithStack(identityProviderListErr)
			}

			for _, identityProvider := range identityProvidersList.Slice() {
				if identityProvider.Name() == openIDIdpInfo.Name {
					return identityProvider.ID(), nil
				}
			}
		}
		return "", errors.WithStack(createIdentityProviderErr)
	}
	return createdIdentityProvider.ID(), nil
}

func buildIdentityProvider(idpInfo types.OpenIDIdentityProviderInfo) (*clustersmgmtv1.IdentityProvider, error) {
	openIdentityBuilder := clustersmgmtv1.NewOpenIDIdentityProvider().
		ClientID(idpInfo.ClientID).
		ClientSecret(idpInfo.ClientSecret).
		Claims(clustersmgmtv1.NewOpenIDClaims().
			Email("email").
			PreferredUsername("preferred_username").
			Name("last_name", "preferred_username")).
		Issuer(idpInfo.Issuer)

	identityProviderBuilder := clustersmgmtv1.NewIdentityProvider().
		Type("OpenIDIdentityProvider").
		MappingMethod(clustersmgmtv1.IdentityProviderMappingMethodClaim).
		OpenID(openIdentityBuilder).
		Name(idpInfo.Name)

	identityProvider, idpBuildErr := identityProviderBuilder.Build()
	if idpBuildErr != nil {
		return nil, fmt.Errorf("building identity provider: %w", idpBuildErr)
	}

	return identityProvider, nil
}
