package config

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gopkg.in/yaml.v2"
)

// DataplaneClusterConfig ...
type DataplaneClusterConfig struct {
	OpenshiftVersion   string `json:"cluster_openshift_version"`
	ComputeMachineType string `json:"cluster_compute_machine_type"`
	// Possible values are:
	// 'manual' to use OSD Cluster configuration file,
	// 'auto' to use dynamic scaling
	// 'none' to disabled scaling all together, useful in testing
	DataPlaneClusterScalingType           string `json:"dataplane_cluster_scaling_type"`
	DataPlaneClusterConfigFile            string `json:"dataplane_cluster_config_file"`
	ReadOnlyUserList                      userv1.OptionalNames
	ReadOnlyUserListFile                  string
	ClusterConfig                         *ClusterConfig `json:"clusters_config"`
	EnableReadyDataPlaneClustersReconcile bool           `json:"enable_ready_dataplane_clusters_reconcile"`
}

// ManualScaling ...
const (
	// ManualScaling is the manual DataPlaneClusterScalingType via the configuration file
	ManualScaling string = "manual"
	// AutoScaling is the automatic DataPlaneClusterScalingType depending on cluster capacity as reported by the Agent Operator
	AutoScaling string = "auto"
	// NoScaling disables cluster scaling. This is useful in testing
	NoScaling string = "none"
)

// NewDataplaneClusterConfig ...
func NewDataplaneClusterConfig() *DataplaneClusterConfig {
	return &DataplaneClusterConfig{
		OpenshiftVersion:                      "",
		ComputeMachineType:                    "m5.2xlarge",
		DataPlaneClusterConfigFile:            "config/dataplane-cluster-configuration.yaml",
		ReadOnlyUserListFile:                  "config/read-only-user-list.yaml",
		DataPlaneClusterScalingType:           ManualScaling,
		ClusterConfig:                         &ClusterConfig{},
		EnableReadyDataPlaneClustersReconcile: true,
	}
}

// ManualCluster manual cluster configuration
type ManualCluster struct {
	Name                  string                  `yaml:"name"`
	ClusterID             string                  `yaml:"cluster_id"`
	CloudProvider         string                  `yaml:"cloud_provider"`
	Region                string                  `yaml:"region"`
	MultiAZ               bool                    `yaml:"multi_az"`
	Schedulable           bool                    `yaml:"schedulable"`
	CentralInstanceLimit  int                     `yaml:"central_instance_limit"`
	Status                api.ClusterStatus       `yaml:"status"`
	ProviderType          api.ClusterProviderType `yaml:"provider_type"`
	ClusterDNS            string                  `yaml:"cluster_dns"`
	SupportedInstanceType string                  `yaml:"supported_instance_type"`
}

// UnmarshalYAML ...
func (c *ManualCluster) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type t ManualCluster
	temp := t{
		Status:                api.ClusterProvisioning,
		ProviderType:          api.ClusterProviderOCM,
		ClusterDNS:            "",
		SupportedInstanceType: api.AllInstanceTypeSupport.String(), // by default support both instance type
	}
	err := unmarshal(&temp)
	if err != nil {
		return err
	}
	*c = ManualCluster(temp)
	if c.ClusterID == "" {
		return fmt.Errorf("cluster_id is empty")
	}

	if c.ProviderType == api.ClusterProviderStandalone {
		if c.ClusterDNS == "" {
			return errors.Errorf("Standalone cluster with id %s does not have the cluster dns field provided", c.ClusterID)
		}

		if c.Name == "" {
			return errors.Errorf("Standalone cluster with id %s does not have the name field provided", c.ClusterID)
		}

		if c.Status != api.ClusterProvisioning && c.Status != api.ClusterProvisioned && c.Status != api.ClusterReady {
			// Force to cluster provisioning status as we do not want to call StandaloneProvider to create the cluster.
			c.Status = api.ClusterProvisioning
		}
	}

	if c.SupportedInstanceType == "" {
		c.SupportedInstanceType = api.AllInstanceTypeSupport.String()
	}
	return nil
}

// ClusterList ...
type ClusterList []ManualCluster

// ClusterConfig ...
type ClusterConfig struct {
	clusterList      ClusterList
	clusterConfigMap map[string]ManualCluster
}

// NewClusterConfig ...
func NewClusterConfig(clusters ClusterList) *ClusterConfig {
	clusterMap := make(map[string]ManualCluster)
	for _, c := range clusters {
		clusterMap[c.ClusterID] = c
	}
	return &ClusterConfig{
		clusterList:      clusters,
		clusterConfigMap: clusterMap,
	}
}

// GetCapacityForRegion ...
func (conf *ClusterConfig) GetCapacityForRegion(region string) int {
	var capacity = 0
	for _, cluster := range conf.clusterList {
		if cluster.Region == region {
			capacity += cluster.CentralInstanceLimit
		}
	}
	return capacity
}

// IsNumberOfCentralWithinClusterLimit ...
func (conf *ClusterConfig) IsNumberOfCentralWithinClusterLimit(clusterID string, count int) bool {
	if _, exist := conf.clusterConfigMap[clusterID]; exist {
		limit := conf.clusterConfigMap[clusterID].CentralInstanceLimit
		return limit == -1 || count <= limit
	}
	return true
}

// IsClusterSchedulable ...
func (conf *ClusterConfig) IsClusterSchedulable(clusterID string) bool {
	if _, exist := conf.clusterConfigMap[clusterID]; exist {
		return conf.clusterConfigMap[clusterID].Schedulable
	}
	return true
}

// GetClusterSupportedInstanceType ...
func (conf *ClusterConfig) GetClusterSupportedInstanceType(clusterID string) (string, bool) {
	manualCluster, exist := conf.clusterConfigMap[clusterID]
	return manualCluster.SupportedInstanceType, exist
}

// ExcessClusters ...
func (conf *ClusterConfig) ExcessClusters(clusterList map[string]api.Cluster) []string {
	var res []string

	for clusterID, v := range clusterList {
		if _, exist := conf.clusterConfigMap[clusterID]; !exist {
			res = append(res, v.ClusterID)
		}
	}
	return res
}

// GetManualClusters ...
func (conf *ClusterConfig) GetManualClusters() []ManualCluster {
	return conf.clusterList
}

// MissingClusters ...
func (conf *ClusterConfig) MissingClusters(clusterMap map[string]api.Cluster) []ManualCluster {
	var res []ManualCluster

	// ensure the order
	for _, p := range conf.clusterList {
		if _, exists := clusterMap[p.ClusterID]; !exists {
			res = append(res, p)
		}
	}
	return res
}

// ExistingClusters produces the subset of clusters which do already exist for fleet-manager.
func (conf *ClusterConfig) ExistingClusters(clusterMap map[string]api.Cluster) []ManualCluster {
	var res []ManualCluster

	// ensure the order
	for _, p := range conf.clusterList {
		if _, exists := clusterMap[p.ClusterID]; exists {
			res = append(res, p)
		}
	}
	return res
}

// IsDataPlaneManualScalingEnabled ...
func (c *DataplaneClusterConfig) IsDataPlaneManualScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == ManualScaling
}

// IsDataPlaneAutoScalingEnabled ...
func (c *DataplaneClusterConfig) IsDataPlaneAutoScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == AutoScaling
}

// IsReadyDataPlaneClustersReconcileEnabled ...
func (c *DataplaneClusterConfig) IsReadyDataPlaneClustersReconcileEnabled() bool {
	return c.EnableReadyDataPlaneClustersReconcile
}

// AddFlags ...
func (c *DataplaneClusterConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.OpenshiftVersion, "cluster-openshift-version", c.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&c.ComputeMachineType, "cluster-compute-machine-type", c.ComputeMachineType, "The compute machine type")
	fs.StringVar(&c.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", c.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&c.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", c.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters. Its value should be either 'none' for no scaling, 'manual' or 'auto'.")
	fs.StringVar(&c.ReadOnlyUserListFile, "read-only-user-list-file", c.ReadOnlyUserListFile, "File contains a list of users with read-only permissions to data plane clusters")
	fs.BoolVar(&c.EnableReadyDataPlaneClustersReconcile, "enable-ready-dataplane-clusters-reconcile", c.EnableReadyDataPlaneClustersReconcile, "Enables reconciliation for data plane clusters in the 'Ready' state")
}

// ReadFiles ...
func (c *DataplaneClusterConfig) ReadFiles() error {
	if c.IsDataPlaneManualScalingEnabled() {
		list, err := readDataPlaneClusterConfig(c.DataPlaneClusterConfigFile)
		if err == nil {
			c.ClusterConfig = NewClusterConfig(list)
		} else {
			return err
		}
	}

	err := readOnlyUserListFile(c.ReadOnlyUserListFile, &c.ReadOnlyUserList)
	if err != nil {
		return err
	}

	return nil
}

func readDataPlaneClusterConfig(file string) (ClusterList, error) {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading data plane cluster config file: %w", err)
	}

	c := struct {
		ClusterList ClusterList `yaml:"clusters"`
	}{}

	if err = yaml.Unmarshal([]byte(fileContents), &c); err != nil {
		return nil, fmt.Errorf("reading data plane cluster config file: %w", err)
	}
	return c.ClusterList, nil
}

// FindClusterNameByClusterID ...
func (c *DataplaneClusterConfig) FindClusterNameByClusterID(clusterID string) string {
	for _, cluster := range c.ClusterConfig.clusterList {
		if cluster.ClusterID == clusterID {
			return cluster.Name
		}
	}
	return ""
}

// Read the read-only users in the file into the read-only user list config
func readOnlyUserListFile(file string, val *userv1.OptionalNames) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading read-only user list file: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), val)
	if err != nil {
		return fmt.Errorf("reading read-only user list file: %w", err)
	}
	return nil
}
