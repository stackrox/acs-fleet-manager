package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/pkg/errors"
	fleetmanagererrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
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
	SkipScheduling     bool          `json:"skip_scheduling"`
	// the provider type for the cluster, e.g. OCM, AWS, GCP, Standalone etc
	ProviderType ClusterProviderType `json:"provider_type"`
	// store the provider-specific information that can be used to managed the openshift/k8s cluster
	ProviderSpec JSON `json:"provider_spec"`
	// store the specs of the openshift/k8s cluster which can be used to access the cluster
	ClusterSpec JSON `json:"cluster_spec"`
	// List of available central operator versions in the cluster. Content MUST be stored
	// with the versions sorted in ascending order as a JSON. See
	// CentralOperatorVersionNumberPartRegex for details on the expected central operator version
	// format. See the CentralOperatorVersion data type for the format of JSON stored. Use the
	// `SetAvailableCentralOperatorVersions` helper method to ensure the correct order is set.
	// Latest position in the list is considered the newest available version.
	AvailableCentralOperatorVersions JSON `json:"available_central_operator_versions"`
	// SupportedInstanceType holds information on what kind of instances types can be provisioned on this cluster.
	// A cluster can support two kinds of instance types: 'eval', 'standard' or both in this case it will be a comma separated list of instance types e.g 'standard,eval'.
	SupportedInstanceType string `json:"supported_instance_type"`
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

// CentralOperatorVersion ...
type CentralOperatorVersion struct {
	Version         string           `json:"version"`
	Ready           bool             `json:"ready"`
	CentralVersions []CentralVersion `json:"centralVersions" yaml:"central_versions"`
}

// CentralVersion ...
type CentralVersion struct {
	Version string `json:"version"`
}

// Compare ...
func (s *CentralVersion) Compare(other CentralVersion) (int, error) {
	return buildAwareSemanticVersioningCompare(s.Version, other.Version)
}

// CentralOperatorVersionNumberPartRegex contains the regular expression needed to
// extract the semver version number for a CentralOperatorVersion. CentralOperatorVersion
// follows the format of: prefix_string-X.Y.Z-B where X,Y,Z,B are numbers
var CentralOperatorVersionNumberPartRegex = regexp.MustCompile(`\d+\.\d+\.\d+-\d+$`)

// Compare returns compares s.Version with other.Version comparing the version
// number suffix specified there using CentralOperatorVersionNumberPartRegex to extract
// the version number. If s.Version is smaller than other.Version a -1 is returned.
// If s.Version is equal than other.Version 0 is returned. If s.Version is greater
// than other.Version 1 is returned. If there is an error during the comparison
// an error is returned
func (s *CentralOperatorVersion) Compare(other CentralOperatorVersion) (int, error) {
	v1VersionNumber := CentralOperatorVersionNumberPartRegex.FindString(s.Version)
	if v1VersionNumber == "" {
		return 0, fmt.Errorf("'%s' does not follow expected Central Operator Version format", s.Version)
	}

	v2VersionNumber := CentralOperatorVersionNumberPartRegex.FindString(other.Version)
	if v2VersionNumber == "" {
		return 0, fmt.Errorf("'%s' does not follow expected Central Operator Version format", s.Version)
	}

	return buildAwareSemanticVersioningCompare(v1VersionNumber, v2VersionNumber)
}

// CompareBuildAwareSemanticVersions ...
func CompareBuildAwareSemanticVersions(v1, v2 string) (int, error) {
	return buildAwareSemanticVersioningCompare(v1, v2)
}

// CompareSemanticVersionsMajorAndMinor ...
func CompareSemanticVersionsMajorAndMinor(current, desired string) (int, error) {
	return checkIfMinorDowngrade(current, desired)
}

// DeepCopy ...
func (s *CentralOperatorVersion) DeepCopy() *CentralOperatorVersion {
	res := *s
	res.CentralVersions = nil

	if s.CentralVersions != nil {
		centralVersionsCopy := make([]CentralVersion, len(s.CentralVersions))
		copy(centralVersionsCopy, s.CentralVersions)
		res.CentralVersions = centralVersionsCopy
	}

	return &res
}

// GetAvailableAndReadyCentralOperatorVersions returns the cluster's list of available
// and ready versions or an error. An empty list is returned if there are no
// available and ready versions
func (cluster *Cluster) GetAvailableAndReadyCentralOperatorVersions() ([]CentralOperatorVersion, error) {
	centralOperatorVersions, err := cluster.GetAvailableCentralOperatorVersions()
	if err != nil {
		return nil, err
	}

	res := []CentralOperatorVersion{}
	for _, val := range centralOperatorVersions {
		if val.Ready {
			res = append(res, val)
		}
	}
	return res, nil
}

// GetAvailableCentralOperatorVersions returns the cluster's list of available central operator
// versions or an error. An empty list is returned if there are no versions.
// This returns the available versions in the cluster independently on whether
// they are ready or not. If you want to only get the available and ready
// versions use the GetAvailableAndReadyCentralOperatorVersions method
func (cluster *Cluster) GetAvailableCentralOperatorVersions() ([]CentralOperatorVersion, error) {
	versions := []CentralOperatorVersion{}
	if cluster.AvailableCentralOperatorVersions == nil {
		return versions, nil
	}

	err := json.Unmarshal(cluster.AvailableCentralOperatorVersions, &versions)
	if err != nil {
		return nil, fmt.Errorf("getting available central operator versions: %w", err)
	}

	return versions, nil
}

// CentralOperatorVersionsDeepSort returns a sorted copy of the provided CentralOperatorVersions
// in the versions slice. The following elements are sorted in ascending order:
// - The central operator versions
// - For each central operator version, their Central Versions
func CentralOperatorVersionsDeepSort(versions []CentralOperatorVersion) ([]CentralOperatorVersion, error) {
	if versions == nil {
		return versions, nil
	}
	if len(versions) == 0 {
		return []CentralOperatorVersion{}, nil
	}

	var versionsToSet []CentralOperatorVersion
	for idx := range versions {
		version := &versions[idx]
		copiedCentralOperatorVersion := version.DeepCopy()
		versionsToSet = append(versionsToSet, *copiedCentralOperatorVersion)
	}

	var errors fleetmanagererrors.ErrorList

	sort.Slice(versionsToSet, func(i, j int) bool {
		res, err := versionsToSet[i].Compare(versionsToSet[j])
		if err != nil {
			errors = append(errors, err)
		}
		return res == -1
	})

	if errors != nil {
		return nil, errors
	}

	for idx := range versionsToSet {

		sort.Slice(versionsToSet[idx].CentralVersions, func(i, j int) bool {
			res, err := versionsToSet[idx].CentralVersions[i].Compare(versionsToSet[idx].CentralVersions[j])
			if err != nil {
				errors = append(errors, err)
			}
			return res == -1
		})

		if errors != nil {
			return nil, errors
		}

		if errors != nil {
			return nil, errors
		}
	}

	return versionsToSet, nil
}

// SetAvailableCentralOperatorVersions sets the cluster's list of available central operator
// versions. The list of versions is always stored in version ascending order,
// with all versions deeply sorted (central operator versions, central versions ...).
// If availableCentralOperatorVersions is nil an empty list is set. See
// CentralOperatorVersionNumberPartRegex for details on the expected central operator version
// format
func (cluster *Cluster) SetAvailableCentralOperatorVersions(availableCentralOperatorVersions []CentralOperatorVersion) error {
	sortedVersions, err := CentralOperatorVersionsDeepSort(availableCentralOperatorVersions)
	if err != nil {
		return err
	}
	if sortedVersions == nil {
		sortedVersions = []CentralOperatorVersion{}
	}
	v, err := json.Marshal(sortedVersions)
	if err != nil {
		return fmt.Errorf("marshalling sorted versions: %w", err)
	}
	cluster.AvailableCentralOperatorVersions = v
	return nil
}
