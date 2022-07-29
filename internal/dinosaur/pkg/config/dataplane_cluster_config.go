package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/constants"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	userv1 "github.com/openshift/api/user/v1"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// DataplaneClusterConfig ...
type DataplaneClusterConfig struct {
	OpenshiftVersion             string `json:"cluster_openshift_version"`
	ComputeMachineType           string `json:"cluster_compute_machine_type"`
	ImagePullDockerConfigContent string `json:"image_pull_docker_config_content"`
	ImagePullDockerConfigFile    string `json:"image_pull_docker_config_file"`
	// Possible values are:
	// 'manual' to use OSD Cluster configuration file,
	// 'auto' to use dynamic scaling
	// 'none' to disabled scaling all together, useful in testing
	DataPlaneClusterScalingType string `json:"dataplane_cluster_scaling_type"`
	DataPlaneClusterConfigFile  string `json:"dataplane_cluster_config_file"`
	ReadOnlyUserList            userv1.OptionalNames
	ReadOnlyUserListFile        string
	// TODO ROX-11294 adjust or drop sre user list
	SREUsers                              userv1.OptionalNames
	ClusterConfig                         *ClusterConfig `json:"clusters_config"`
	EnableReadyDataPlaneClustersReconcile bool           `json:"enable_ready_dataplane_clusters_reconcile"`
	Kubeconfig                            string         `json:"kubeconfig"`
	RawKubernetesConfig                   *clientcmdapi.Config
	CentralOperatorOLMConfig              OperatorInstallationConfig `json:"dinosaur_operator_olm_config"`
	FleetshardOperatorOLMConfig           OperatorInstallationConfig `json:"fleetshard_operator_olm_config"`
}

// OperatorInstallationConfig ...
type OperatorInstallationConfig struct {
	Namespace              string `json:"namespace"`
	IndexImage             string `json:"index_image"`
	CatalogSourceNamespace string `json:"catalog_source_namespace"`
	Package                string `json:"package"`
	SubscriptionChannel    string `json:"subscription_channel"`
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

func getDefaultKubeconfig() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".kube", "config")
}

// NewDataplaneClusterConfig ...
func NewDataplaneClusterConfig() *DataplaneClusterConfig {
	return &DataplaneClusterConfig{
		OpenshiftVersion:                      "",
		ComputeMachineType:                    "m5.2xlarge",
		ImagePullDockerConfigContent:          "",
		ImagePullDockerConfigFile:             "secrets/image-pull.dockerconfigjson",
		DataPlaneClusterConfigFile:            "config/dataplane-cluster-configuration.yaml",
		ReadOnlyUserListFile:                  "config/read-only-user-list.yaml",
		DataPlaneClusterScalingType:           ManualScaling,
		ClusterConfig:                         &ClusterConfig{},
		EnableReadyDataPlaneClustersReconcile: true,
		Kubeconfig:                            getDefaultKubeconfig(),
		CentralOperatorOLMConfig: OperatorInstallationConfig{
			IndexImage:             "quay.io/osd-addons/managed-central:production-82b42db",
			CatalogSourceNamespace: "openshift-marketplace",
			Namespace:              constants.CentralOperatorNamespace,
			SubscriptionChannel:    "alpha",
			Package:                "managed-central",
		},
		FleetshardOperatorOLMConfig: OperatorInstallationConfig{
			IndexImage:             "quay.io/osd-addons/fleetshard-operator:production-82b42db",
			CatalogSourceNamespace: "openshift-marketplace",
			Namespace:              constants.FleetShardOperatorNamespace,
			SubscriptionChannel:    "alpha",
			Package:                "fleetshard-operator",
		},
	}
}

// ManualCluster manual cluster configuration
type ManualCluster struct {
	Name                             string                       `yaml:"name"`
	ClusterID                        string                       `yaml:"cluster_id"`
	CloudProvider                    string                       `yaml:"cloud_provider"`
	Region                           string                       `yaml:"region"`
	MultiAZ                          bool                         `yaml:"multi_az"`
	Schedulable                      bool                         `yaml:"schedulable"`
	CentralInstanceLimit             int                          `yaml:"central_instance_limit"`
	Status                           api.ClusterStatus            `yaml:"status"`
	ProviderType                     api.ClusterProviderType      `yaml:"provider_type"`
	ClusterDNS                       string                       `yaml:"cluster_dns"`
	SupportedInstanceType            string                       `yaml:"supported_instance_type"`
	AvailableCentralOperatorVersions []api.CentralOperatorVersion `yaml:"available_central_operator_versions"`
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

// IsNumberOfDinosaurWithinClusterLimit ...
func (conf *ClusterConfig) IsNumberOfDinosaurWithinClusterLimit(clusterID string, count int) bool {
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
	fs.StringVar(&c.ImagePullDockerConfigFile, "image-pull-docker-config-file", c.ImagePullDockerConfigFile, "The file that contains the docker config content for pulling MK operator images on clusters")
	fs.StringVar(&c.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", c.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&c.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", c.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters. Its value should be either 'none' for no scaling, 'manual' or 'auto'.")
	fs.StringVar(&c.ReadOnlyUserListFile, "read-only-user-list-file", c.ReadOnlyUserListFile, "File contains a list of users with read-only permissions to data plane clusters")
	fs.BoolVar(&c.EnableReadyDataPlaneClustersReconcile, "enable-ready-dataplane-clusters-reconcile", c.EnableReadyDataPlaneClustersReconcile, "Enables reconciliation for data plane clusters in the 'Ready' state")
	fs.StringVar(&c.Kubeconfig, "kubeconfig", c.Kubeconfig, "A path to kubeconfig file used for communication with standalone clusters")
	fs.StringVar(&c.CentralOperatorOLMConfig.CatalogSourceNamespace, "central-operator-cs-namespace", c.CentralOperatorOLMConfig.CatalogSourceNamespace, "Central operator catalog source namespace.")
	fs.StringVar(&c.CentralOperatorOLMConfig.IndexImage, "central-operator-index-image", c.CentralOperatorOLMConfig.IndexImage, "Central operator index image")
	fs.StringVar(&c.CentralOperatorOLMConfig.Namespace, "central-operator-namespace", c.CentralOperatorOLMConfig.Namespace, "Central operator namespace")
	fs.StringVar(&c.CentralOperatorOLMConfig.Package, "central-operator-package", c.CentralOperatorOLMConfig.Package, "Central operator package")
	fs.StringVar(&c.CentralOperatorOLMConfig.SubscriptionChannel, "central-operator-sub-channel", c.CentralOperatorOLMConfig.SubscriptionChannel, "Central operator subscription channel")
	fs.StringVar(&c.FleetshardOperatorOLMConfig.CatalogSourceNamespace, "fleetshard-operator-cs-namespace", c.FleetshardOperatorOLMConfig.CatalogSourceNamespace, "fleetshard operator catalog source namespace.")
	fs.StringVar(&c.FleetshardOperatorOLMConfig.IndexImage, "fleetshard-operator-index-image", c.FleetshardOperatorOLMConfig.IndexImage, "fleetshard operator index image")
	fs.StringVar(&c.FleetshardOperatorOLMConfig.Namespace, "fleetshard-operator-namespace", c.FleetshardOperatorOLMConfig.Namespace, "fleetshard operator namespace")
	fs.StringVar(&c.FleetshardOperatorOLMConfig.Package, "fleetshard-operator-package", c.FleetshardOperatorOLMConfig.Package, "fleetshard operator package")
	fs.StringVar(&c.FleetshardOperatorOLMConfig.SubscriptionChannel, "fleetshard-operator-sub-channel", c.FleetshardOperatorOLMConfig.SubscriptionChannel, "fleetshard operator subscription channel")
}

// ReadFiles ...
func (c *DataplaneClusterConfig) ReadFiles() error {
	if c.ImagePullDockerConfigContent == "" && c.ImagePullDockerConfigFile != "" {
		err := shared.ReadFileValueString(c.ImagePullDockerConfigFile, &c.ImagePullDockerConfigContent)
		if err != nil {
			return fmt.Errorf("reading image pull docker config file: %w", err)
		}
	}

	if c.IsDataPlaneManualScalingEnabled() {
		list, err := readDataPlaneClusterConfig(c.DataPlaneClusterConfigFile)
		if err == nil {
			c.ClusterConfig = NewClusterConfig(list)
		} else {
			return err
		}

		// read kubeconfig and validate standalone clusters are in kubeconfig context
		for _, cluster := range c.ClusterConfig.clusterList {
			if cluster.ProviderType != api.ClusterProviderStandalone {
				continue
			}
			// make sure we only read kubeconfig once
			if c.RawKubernetesConfig == nil {
				err = c.readKubeconfig()
				if err != nil {
					return err
				}
			}
			validationErr := validateClusterIsInKubeconfigContext(*c.RawKubernetesConfig, cluster)
			if validationErr != nil {
				return validationErr
			}
		}
	}

	err := readOnlyUserListFile(c.ReadOnlyUserListFile, &c.ReadOnlyUserList)
	if err != nil {
		return err
	}

	return nil
}

func (c *DataplaneClusterConfig) readKubeconfig() error {
	_, err := os.Stat(c.Kubeconfig)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.Errorf("The kubeconfig file %s does not exist", c.Kubeconfig)
		}
		return fmt.Errorf("retrieving FileInfo for kubeconfig: %w", err)
	}
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{Precedence: []string{c.Kubeconfig}},
		&clientcmd.ConfigOverrides{})
	rawConfig, err := config.RawConfig()
	if err != nil {
		return fmt.Errorf("reading kubeconfig: %w", err)
	}
	c.RawKubernetesConfig = &rawConfig
	return nil
}

func validateClusterIsInKubeconfigContext(rawConfig clientcmdapi.Config, cluster ManualCluster) error {
	if _, found := rawConfig.Contexts[cluster.Name]; found {
		return nil
	}
	return errors.Errorf("standalone cluster with ID: %s, and Name %s not in kubeconfig context", cluster.ClusterID, cluster.Name)
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

// Read the dinosaur-sre users from the file into the dinosaur-sre user list config
func readDinosaurSREUserFile(file string, val *userv1.OptionalNames) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading SRE user list file: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), val)
	if err != nil {
		return fmt.Errorf("reading SRE user list file: %w", err)
	}
	return nil
}
