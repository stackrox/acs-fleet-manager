package operator

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"sigs.k8s.io/yaml"
)

func parseConfig(content []byte) (OperatorConfigs, error) {
	var out OperatorConfigs
	err := yaml.Unmarshal(content, &out)
	if err != nil {
		return OperatorConfigs{}, fmt.Errorf("unmarshalling operator config %w", err)
	}
	return out, nil
}

// GetConfig returns the rhacs operator configurations
func GetConfig() OperatorConfigs {
	// TODO: Read config from GitOps configuration
	glog.Error("Reading RHACS Operator GitOps configuration not implemented yet")
	return OperatorConfigs{}
}

// Validate validates the operator configuration and can be used in different life-cycle stages like runtime and deploy time.
func Validate(configs OperatorConfigs) []error {
	var errors []error
	manager := ACSOperatorManager{
		// TODO: align config URL with fleetshard-sync default
		DefaultBaseCRDURL: "https://raw.githubusercontent.com/stackrox/stackrox/%s/operator/bundle/manifests/",
	}

	// TODO: how to run this locally with the same config as fleet-manager?
	manifests, err := manager.RenderChart(configs)
	if err != nil {
		errors = append(errors, fmt.Errorf("could not render operator helm charts, got invalid configuration: %s", err.Error()))
	} else if len(manifests) == 0 {
		errors = append(errors, fmt.Errorf("operator chart rendering succeed, but no manifests rendered"))
	}

	return errors
}

// CRDConfig represents the crd to be installed in the data-plane cluster. The CRD is downloaded automatically
// from the base URL. It takes a GitRef to resolve a GitHub link to the CRD definition.
type CRDConfig struct {
	BaseURL string `json:"baseURL,omitempty"`
	GitRef  string `json:"gitRef"`
}

// OperatorConfigs represents all operators and the CRD which should be installed in a data-plane cluster.
type OperatorConfigs struct {
	CRD     CRDConfig        `json:"crd"`
	Configs []OperatorConfig `json:"operators"`
}

// OperatorConfig represents the configuration of an operator.
type OperatorConfig struct {
	Image      string `json:"image"`
	GitRef     string `json:"gitRef"`
	HelmValues string `json:"helmValues,omitempty"`
}

// ToAPIResponse transforms the config to an private API response.
func (o OperatorConfigs) ToAPIResponse() private.RhacsOperatorConfigs {
	apiConfigs := private.RhacsOperatorConfigs{
		CRD: private.RhacsOperatorConfigsCrd{
			GitRef:  o.CRD.GitRef,
			BaseURL: o.CRD.BaseURL,
		},
	}

	for _, config := range o.Configs {
		apiConfigs.RHACSOperatorConfigs = append(apiConfigs.RHACSOperatorConfigs, config.ToAPIResponse())
	}
	return apiConfigs
}

// ToAPIResponse converts the internal OperatorConfig to the openapi generated private.RhacsOperatorConfig type.
func (o OperatorConfig) ToAPIResponse() private.RhacsOperatorConfig {
	return private.RhacsOperatorConfig{
		Image:      o.Image,
		GitRef:     o.GitRef,
		HelmValues: o.HelmValues,
	}
}

// FromAPIResponse converts an openapi generated model to the internal OperatorConfigs type
func FromAPIResponse(config private.RhacsOperatorConfigs) OperatorConfigs {
	var operatorConfigs []OperatorConfig
	for _, apiConfig := range config.RHACSOperatorConfigs {
		config := OperatorConfig{
			Image:      apiConfig.Image,
			GitRef:     apiConfig.GitRef,
			HelmValues: apiConfig.HelmValues,
		}
		operatorConfigs = append(operatorConfigs, config)
	}

	return OperatorConfigs{
		Configs: operatorConfigs,
		CRD: CRDConfig{
			GitRef:  config.CRD.GitRef,
			BaseURL: config.CRD.BaseURL,
		},
	}
}
