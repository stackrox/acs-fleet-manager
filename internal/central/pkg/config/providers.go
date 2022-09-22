package config

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"gopkg.in/yaml.v2"
)

// InstanceType ...
type InstanceType types.DinosaurInstanceType

// InstanceTypeMap ...
type InstanceTypeMap map[string]InstanceTypeConfig

// InstanceTypeConfig ...
type InstanceTypeConfig struct {
	Limit *int `yaml:"limit,omitempty"`
}

// AsSlice Returns a region's supported instance type list as a slice
func (itl InstanceTypeMap) AsSlice() []string {
	instanceTypeList := []string{}

	for k := range itl {
		instanceTypeList = append(instanceTypeList, k)
	}

	return instanceTypeList
}

// Region ...
type Region struct {
	Name                   string          `yaml:"name"`
	Default                bool            `yaml:"default"`
	SupportedInstanceTypes InstanceTypeMap `yaml:"supported_instance_type"`
}

// IsInstanceTypeSupported ...
func (r Region) IsInstanceTypeSupported(instanceType InstanceType) bool {
	for k := range r.SupportedInstanceTypes {
		if k == string(instanceType) {
			return true
		}
	}
	return false
}

// RegionList ...
type RegionList []Region

// GetByName ...
func (rl RegionList) GetByName(regionName string) (Region, bool) {
	for _, r := range rl {
		if r.Name == regionName {
			return r, true
		}
	}
	return Region{}, false
}

// String ...
func (rl RegionList) String() string {
	var names []string
	for _, r := range rl {
		names = append(names, r.Name)
	}
	return fmt.Sprint(names)
}

// Provider ...
type Provider struct {
	Name    string     `json:"name"`
	Default bool       `json:"default"`
	Regions RegionList `json:"regions"`
}

// ProviderList ...
type ProviderList []Provider

// GetByName ...
func (pl ProviderList) GetByName(providerName string) (Provider, bool) {
	for _, p := range pl {
		if p.Name == providerName {
			return p, true
		}
	}
	return Provider{}, false
}

// String ...
func (pl ProviderList) String() string {
	var names []string
	for _, p := range pl {
		names = append(names, p.Name)
	}
	return fmt.Sprint(names)
}

// ProviderConfiguration ...
type ProviderConfiguration struct {
	SupportedProviders ProviderList `yaml:"supported_providers"`
}

// ProviderConfig ...
type ProviderConfig struct {
	ProvidersConfig     ProviderConfiguration `json:"providers"`
	ProvidersConfigFile string                `json:"providers_config_file"`
}

// NewSupportedProvidersConfig ...
func NewSupportedProvidersConfig() *ProviderConfig {
	return &ProviderConfig{
		ProvidersConfigFile: "config/provider-configuration.yaml",
	}
}

var _ environments.ServiceValidator = &ProviderConfig{}

// Validate ...
func (c *ProviderConfig) Validate() error {
	providerDefaultCount := 0
	for _, p := range c.ProvidersConfig.SupportedProviders {
		if err := p.Validate(); err != nil {
			return err
		}
		if p.Default {
			providerDefaultCount++
		}
	}
	if providerDefaultCount != 1 {
		return fmt.Errorf("expected 1 default provider in provider list, got %d", providerDefaultCount)
	}
	return nil
}

// Validate ...
func (provider Provider) Validate() error {
	defaultCount := 0
	for _, p := range provider.Regions {
		if p.Default {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		return fmt.Errorf("expected 1 default region in provider %s, got %d", provider.Name, defaultCount)
	}
	return nil
}

// AddFlags ...
func (c *ProviderConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ProvidersConfigFile, "providers-config-file", c.ProvidersConfigFile, "SupportedProviders configuration file")
}

// ReadFiles ...
func (c *ProviderConfig) ReadFiles() error {
	return readFileProvidersConfig(c.ProvidersConfigFile, &c.ProvidersConfig)
}

// Read the contents of file into the providers config
func readFileProvidersConfig(file string, val *ProviderConfiguration) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading providers config file: %w", err)
	}
	err = yaml.UnmarshalStrict([]byte(fileContents), val)
	if err != nil {
		return fmt.Errorf("unmarshalling providers config file from YAML: %w", err)
	}
	return nil
}

// GetDefault ...
func (pl ProviderList) GetDefault() (Provider, error) {
	for _, p := range pl {
		if p.Default {
			return p, nil
		}
	}
	return Provider{}, errors.New("no default provider found in list of supported providers")
}

// GetDefaultRegion ...
func (provider Provider) GetDefaultRegion() (Region, error) {
	for _, r := range provider.Regions {
		if r.Default {
			return r, nil
		}
	}
	return Region{}, fmt.Errorf("no default region found for provider %s", provider.Name)
}

// IsRegionSupported ...
func (provider Provider) IsRegionSupported(regionName string) bool {
	_, ok := provider.Regions.GetByName(regionName)
	return ok
}
