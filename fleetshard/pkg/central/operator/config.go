package operator

import (
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"sigs.k8s.io/yaml"
)

const (
	keyDeploymentName                  = "deploymentName"
	keyImage                           = "image"
	keyDisableCentralReconciler        = "disableCentralReconciler"
	keyDisableSecuredClusterReconciler = "disableSecuredClusterReconciler"
	keyCentralLabelSelector            = "centralLabelSelector"
	keySecuredClusterSelector          = "securedClusterLabelSelector"
)

func parseConfig(content []byte) (OperatorConfigs, error) {
	var out OperatorConfigs
	err := yaml.Unmarshal(content, &out)
	if err != nil {
		return OperatorConfigs{}, fmt.Errorf("unmarshalling operator config %w", err)
	}
	return out, nil
}

// OperatorConfigs represents all operators and the CRD which should be installed in a data-plane cluster.
type OperatorConfigs struct {
	CRDURLs []string         `json:"crdUrls"`
	Configs []OperatorConfig `json:"operators"`
}

// OperatorConfig represents the configuration of an operator.
type OperatorConfig map[string]interface{}

// GetDeploymentName returns the deployment name of the operator.
func (o OperatorConfig) GetDeploymentName() string {
	return o.getString(keyDeploymentName)
}

// GetImage returns the image of the operator.
func (o OperatorConfig) GetImage() string {
	return o.getString(keyImage)
}

// GetCentralLabelSelector returns the central label selector.
func (o OperatorConfig) GetCentralLabelSelector() string {
	return o.getString(keyCentralLabelSelector)
}

// GetSecuredClusterLabelSelector returns the secured cluster label selector.
func (o OperatorConfig) GetSecuredClusterLabelSelector() string {
	return o.getString(keySecuredClusterSelector)
}

// GetDisableCentralReconciler returns true if the central reconciler should be disabled.
func (o OperatorConfig) GetDisableCentralReconciler() bool {
	return o.getBool(keyDisableCentralReconciler)
}

// GetDisableSecuredClusterReconciler returns true if the secured cluster reconciler should be disabled.
func (o OperatorConfig) GetDisableSecuredClusterReconciler() bool {
	return o.getBool(keyDisableSecuredClusterReconciler)
}

func (o OperatorConfig) getString(key string) string {
	valIntf, ok := o[key]
	if !ok {
		return ""
	}
	val, ok := valIntf.(string)
	if !ok {
		return ""
	}
	return val
}

func (o OperatorConfig) getBool(key string) bool {
	valIntf, ok := o[key]
	if !ok {
		return false
	}
	val, ok := valIntf.(bool)
	if !ok {
		return false
	}
	return val
}

// ToAPIResponse transforms the config to an private API response.
func (o OperatorConfigs) ToAPIResponse() private.RhacsOperatorConfigs {
	apiConfigs := private.RhacsOperatorConfigs{
		CrdUrls: o.CRDURLs,
	}
	for _, config := range o.Configs {
		apiConfigs.RHACSOperatorConfigs = append(apiConfigs.RHACSOperatorConfigs, config)
	}
	return apiConfigs
}
