package api

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// ClusterStatus ...
type ClusterStatus string

// ClusterProviderType ...
type ClusterProviderType string

// ClusterInstanceTypeSupport ...
type ClusterInstanceTypeSupport string

// String ...
func (k ClusterStatus) String() string {
	return string(k)
}

// String ...
func (k ClusterInstanceTypeSupport) String() string {
	return string(k)
}

// UnmarshalYAML ...
func (k *ClusterStatus) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	switch s {
	case ClusterProvisioning.String():
		*k = ClusterProvisioning
	case ClusterProvisioned.String():
		*k = ClusterProvisioned
	case ClusterReady.String():
		*k = ClusterReady
	default:
		return errors.Errorf("invalid value %s", s)
	}
	return nil
}

// CompareTo - Compare this status with the given status returning an int. The result will be 0 if k==k1, -1 if k < k1, and +1 if k > k1
func (k ClusterStatus) CompareTo(k1 ClusterStatus) int {
	ordinalK := ordinals[k.String()]
	ordinalK1 := ordinals[k1.String()]

	switch {
	case ordinalK == ordinalK1:
		return 0
	case ordinalK > ordinalK1:
		return 1
	default:
		return -1
	}
}

// String ...
func (p ClusterProviderType) String() string {
	return string(p)
}

// UnmarshalYAML ...
func (p *ClusterProviderType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	switch s {
	case ClusterProviderOCM.String():
		*p = ClusterProviderOCM
	case ClusterProviderAwsEKS.String():
		*p = ClusterProviderAwsEKS
	case ClusterProviderStandalone.String():
		*p = ClusterProviderStandalone
	default:
		return errors.Errorf("invalid value %s", s)
	}
	return nil
}

// ClusterAccepted ...
const (
	// The create cluster request has been recorder
	ClusterAccepted ClusterStatus = "cluster_accepted"
	// ClusterProvisioning the underlying ocm cluster is provisioning
	ClusterProvisioning ClusterStatus = "cluster_provisioning"
	// ClusterProvisioned the underlying ocm cluster is provisioned
	ClusterProvisioned ClusterStatus = "cluster_provisioned"
	// ClusterFailed the cluster failed to become ready
	ClusterFailed ClusterStatus = "failed"
	// ClusterReady the cluster is terraformed and ready for central instances
	ClusterReady ClusterStatus = "ready"
	// ClusterDeprovisioning the cluster is empty and can be deprovisioned
	ClusterDeprovisioning ClusterStatus = "deprovisioning"
	// ClusterCleanup the cluster external resources are being removed
	ClusterCleanup ClusterStatus = "cleanup"
	// ClusterWaitingForFleetShardOperator the cluster is waiting for the fleetshard operator to be ready
	ClusterWaitingForFleetShardOperator ClusterStatus = "waiting_for_fleetshard_operator"
	// ClusterFull the cluster is full and cannot accept more Central clusters
	ClusterFull ClusterStatus = "full"
	// ClusterComputeNodeScalingUp the cluster is in the process of scaling up a compute node
	ClusterComputeNodeScalingUp ClusterStatus = "compute_node_scaling_up"

	ClusterProviderOCM        ClusterProviderType = "ocm"
	ClusterProviderAwsEKS     ClusterProviderType = "aws_eks"
	ClusterProviderStandalone ClusterProviderType = "standalone"

	EvalTypeSupport        ClusterInstanceTypeSupport = "eval"
	StandardTypeSupport    ClusterInstanceTypeSupport = "standard"
	AllInstanceTypeSupport ClusterInstanceTypeSupport = "standard,eval"
)

// ordinals - Used to decide if a status comes after or before a given state
var ordinals = map[string]int{
	ClusterAccepted.String():                     0,
	ClusterProvisioning.String():                 10,
	ClusterProvisioned.String():                  20,
	ClusterWaitingForFleetShardOperator.String(): 30,
	ClusterReady.String():                        40,
	ClusterComputeNodeScalingUp.String():         50,
	ClusterDeprovisioning.String():               60,
	ClusterCleanup.String():                      70,
	ClusterFailed.String():                       80,
}

// StatusForValidCluster This represents the valid statuses of a dataplane cluster
var StatusForValidCluster = []string{string(ClusterProvisioning), string(ClusterProvisioned), string(ClusterReady),
	string(ClusterAccepted), string(ClusterWaitingForFleetShardOperator), string(ClusterComputeNodeScalingUp)}

// ClusterDeletionStatuses are statuses of clusters under deletion
var ClusterDeletionStatuses = []string{ClusterCleanup.String(), ClusterDeprovisioning.String()}

// Cluster ...
type Cluster struct {
	Meta
	CloudProvider      string        `json:"cloud_provider"`
	ClusterID          string        `json:"cluster_id" gorm:"uniqueIndex"`
	ExternalID         string        `json:"external_id"`
	MultiAZ            bool          `json:"multi_az"`
	Region             string        `json:"region"`
	Status             ClusterStatus `json:"status" gorm:"index"`
	StatusDetails      string        `json:"status_details" gorm:"-"`
	IdentityProviderID string        `json:"identity_provider_id"`
	ClusterDNS         string        `json:"cluster_dns"`
	// the provider type for the cluster, e.g. OCM, AWS, GCP, Standalone etc
	ProviderType ClusterProviderType `json:"provider_type"`
	// store the provider-specific information that can be used to manage the openshift/k8s cluster
	ProviderSpec JSON `json:"provider_spec"`
	// store the specs of the openshift/k8s cluster which can be used to access the cluster
	ClusterSpec JSON `json:"cluster_spec"`
	// SupportedInstanceType holds information on what kind of instances types can be provisioned on this cluster.
	// A cluster can support two kinds of instance types: 'eval', 'standard' or both in this case it will be a comma separated list of instance types e.g 'standard,eval'.
	SupportedInstanceType string `json:"supported_instance_type"`
	// The cluster is "schedulable" if tenants can be placed there.
	Schedulable bool `json:"schedulable"`
	// stores the actual information about the fleetshard addon installed on the cluster
	FleetshardAddonStatus JSON `json:"fleetshard_addon_status"`
}

// ClusterList ...
type ClusterList []*Cluster

// ClusterIndex ...
type ClusterIndex map[string]*Cluster

// Index ...
func (c ClusterList) Index() ClusterIndex {
	index := ClusterIndex{}
	for _, o := range c {
		index[o.ID] = o
	}
	return index
}

// BeforeCreate ...
func (cluster *Cluster) BeforeCreate(tx *gorm.DB) error {
	if cluster.Status == "" {
		cluster.Status = ClusterAccepted
	}

	if cluster.ID == "" {
		cluster.ID = NewID()
	}

	if cluster.SupportedInstanceType == "" {
		cluster.SupportedInstanceType = AllInstanceTypeSupport.String()
	}

	return nil
}
