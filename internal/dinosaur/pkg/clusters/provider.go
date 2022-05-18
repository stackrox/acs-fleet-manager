package clusters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/pkg/errors"
)

//go:generate moq -out provider_moq.go . Provider
type Provider interface {
	// Create using the information provided to request a new OpenShift/k8s cluster from the provider
	Create(request *types.ClusterRequest) (*types.ClusterSpec, error)
	// Delete delete the cluster from the provider
	Delete(spec *types.ClusterSpec) (bool, error)
	// CheckClusterStatus check the status of the cluster. This will be called periodically during cluster provisioning phase to see if the cluster is ready.
	// It should set the status in the returned `ClusterSpec` to either `provisioning`, `ready` or `failed`.
	// If there is additional data that needs to be preserved and passed between checks, add it to the returned `ClusterSpec` and it will be saved to the database and passed into this function again next time it is called.
	CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error)
	// AddIdentityProvider add an identity provider to the cluster
	AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error)
	// ApplyResources apply openshift/k8s resources to the cluster
	ApplyResources(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error)
	// ScaleUp scale the cluster up with the number of additional nodes specified
	ScaleUp(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error)
	// ScaleDown scale the cluster down with the number of nodes specified
	ScaleDown(clusterSpec *types.ClusterSpec, decrement int) (*types.ClusterSpec, error)
	// SetComputeNodes set the number of desired compute nodes for the cluster
	SetComputeNodes(clusterSpec *types.ClusterSpec, numNodes int) (*types.ClusterSpec, error)
	// GetComputeNodes get the number of compute nodes for the cluster
	GetComputeNodes(spec *types.ClusterSpec) (*types.ComputeNodesInfo, error)
	// GetClusterDNS Get the dns of the cluster
	GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error)
	// GetCloudProviders Get the information about supported cloud providers from the cluster provider
	GetCloudProviders() (*types.CloudProviderInfoList, error)
	// GetCloudProviderRegions Get the regions information for the given cloud provider from the cluster provider
	GetCloudProviderRegions(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error)
	// Install the dinosaur operator in a given cluster
	InstallDinosaurOperator(clusterSpec *types.ClusterSpec) (bool, error)
	// Install the cluster logging operator for a given cluster
	InstallFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error)
}

// ProviderFactory used to return an instance of Provider implementation
//go:generate moq -out provider_factory_moq.go . ProviderFactory
type ProviderFactory interface {
	GetProvider(providerType api.ClusterProviderType) (Provider, error)
}

// DefaultProviderFactory the default implementation for ProviderFactory
type DefaultProviderFactory struct {
	providerContainer map[api.ClusterProviderType]Provider
}

func NewDefaultProviderFactory(
	ocmClient ocm.ClusterManagementClient,
	connectionFactory *db.ConnectionFactory,
	ocmConfig *ocm.OCMConfig,
	awsConfig *config.AWSConfig,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
) *DefaultProviderFactory {
	ocmProvider := newOCMProvider(ocmClient, NewClusterBuilder(awsConfig, dataplaneClusterConfig), ocmConfig)
	standaloneProvider := newStandaloneProvider(connectionFactory, dataplaneClusterConfig)
	return &DefaultProviderFactory{
		providerContainer: map[api.ClusterProviderType]Provider{
			api.ClusterProviderStandalone: standaloneProvider,
			api.ClusterProviderOCM:        ocmProvider,
		},
	}
}

func (d *DefaultProviderFactory) GetProvider(providerType api.ClusterProviderType) (Provider, error) {
	if providerType == "" {
		providerType = api.ClusterProviderOCM
	}

	provider, ok := d.providerContainer[providerType]
	if !ok {
		return nil, errors.Errorf("invalid provider type: %v", providerType)
	}

	return provider, nil
}
