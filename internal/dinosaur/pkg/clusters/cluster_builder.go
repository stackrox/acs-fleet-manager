package clusters

import (
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
)

// ClusterNamePrefix a prefix used for new OCM cluster names
const (
	ClusterNamePrefix  = "mk-"
	ComputeMachineType = "m5.2xlarge"
)

// ClusterBuilder NOTE: the current mock generation exports to a _test file, if in the future this should be made public, consider
// moving the type into a ocmtest package.
//go:generate moq -out clusterbuilder_moq.go . ClusterBuilder
// ClusterBuilder wrapper for the OCM-specific builder struct, to allow for mocking.
type ClusterBuilder interface {
	// NewOCMClusterFromCluster create an OCM cluster definition that can be used to create a new cluster with the OCM
	// Cluster Service.
	NewOCMClusterFromCluster(clusterRequest *types.ClusterRequest) (*clustersmgmtv1.Cluster, error)
}

var _ ClusterBuilder = &clusterBuilder{}

// clusterBuilder internal ClusterBuilder implementation.
type clusterBuilder struct {
	// idGenerator generates cluster IDs.
	idGenerator ocm.IDGenerator

	// awsConfig contains aws credentials for use with the OCM cluster service.
	awsConfig *config.AWSConfig

	// dataplaneClusterConfig contains cluster creation configuration.
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

// NewClusterBuilder create a new default implementation of ClusterBuilder.
func NewClusterBuilder(awsConfig *config.AWSConfig, dataplaneClusterConfig *config.DataplaneClusterConfig) ClusterBuilder {
	return &clusterBuilder{
		idGenerator:            ocm.NewIDGenerator(ClusterNamePrefix),
		awsConfig:              awsConfig,
		dataplaneClusterConfig: dataplaneClusterConfig,
	}
}

// NewOCMClusterFromCluster ...
func (r clusterBuilder) NewOCMClusterFromCluster(clusterRequest *types.ClusterRequest) (*clustersmgmtv1.Cluster, error) {
	// pre-req nil checks
	if err := r.validate(); err != nil {
		return nil, err
	}
	if clusterRequest == nil {
		return nil, errors.Errorf("cluster request is nil")
	}

	clusterBuilder := clustersmgmtv1.NewCluster()
	// the name of the cluster must start with a letter, use a standardised prefix to guarentee this.
	clusterBuilder.Name(r.idGenerator.Generate())
	clusterBuilder.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(clusterRequest.CloudProvider))
	clusterBuilder.Region(clustersmgmtv1.NewCloudRegion().ID(clusterRequest.Region))
	// currently only enabled for MultiAZ.
	clusterBuilder.MultiAZ(true)
	if r.dataplaneClusterConfig.OpenshiftVersion != "" {
		clusterBuilder.Version(clustersmgmtv1.NewVersion().ID(r.dataplaneClusterConfig.OpenshiftVersion))
	}
	// setting CCS to always be true for now as this is the only available cluster type within our quota.
	clusterBuilder.CCS(clustersmgmtv1.NewCCS().Enabled(true))

	clusterBuilder.Managed(true)

	// AWS config read from the secrets/aws.* files
	awsBuilder := clustersmgmtv1.NewAWS().AccountID(r.awsConfig.AccountID).AccessKeyID(r.awsConfig.AccessKey).SecretAccessKey(r.awsConfig.SecretAccessKey)
	clusterBuilder.AWS(awsBuilder)

	// Set compute node size
	clusterBuilder.Nodes(clustersmgmtv1.NewClusterNodes().ComputeMachineType(clustersmgmtv1.NewMachineType().ID(r.dataplaneClusterConfig.ComputeMachineType)))

	return clusterBuilder.Build()
}

// validate validate the state of the clusterBuilder struct.
func (r clusterBuilder) validate() error {
	if r.idGenerator == nil {
		return errors.Errorf("idGenerator is not defined")
	}

	if r.awsConfig == nil {
		return errors.Errorf("awsConfig is not defined")
	}

	if r.dataplaneClusterConfig == nil {
		return errors.Errorf("dataplaneClusterConfig is not defined")
	}

	return nil
}
