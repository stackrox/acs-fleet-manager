// Package gitops contains the GitOps configuration.
package gitops

import (
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Config represents the gitops configuration
type Config struct {
	Centrals       CentralsConfig           `json:"centrals"`
	RHACSOperators operator.OperatorConfigs `json:"rhacsOperators"`
}

// CentralsConfig represents the declarative configuration for Central instances defaults and overrides.
type CentralsConfig struct {
	// Overrides are the overrides for Central instances.
	Overrides []CentralOverride `json:"overrides"`
}

// CentralOverride represents the configuration for a Central instance override. The override
// will be applied on top of the default central instance configuration.
// See https://github.com/stackrox/stackrox/blob/master/operator/apis/platform/v1alpha1/overlay_types.go
type CentralOverride struct {
	// InstanceIDs are the Central instance IDs for the override.
	InstanceIDs []string `json:"instanceIds"`
	// Patch is the patch for the override, which will be applied as a strategic merge patch.
	Patch string `json:"patch"`
}

// ValidateConfig validates the GitOps configuration.
func (c *Config) ValidateConfig() field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, c.validateCentralsConfig(field.NewPath("centrals"), c.Centrals)...)
	errs = append(errs, operator.Validate(field.NewPath("rhacsOperators"), c.RHACSOperators)...)
	return errs
}

func (c *Config) validateCentralsConfig(path *field.Path, config CentralsConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, c.validateCentralOverrides(path.Child("overrides"), config.Overrides)...)
	return errs
}

func (c *Config) validateCentralOverrides(path *field.Path, config []CentralOverride) field.ErrorList {
	var errs field.ErrorList
	for i, override := range config {
		errs = append(errs, c.validateCentralOverride(path.Index(i), override)...)
	}
	return errs
}

func (c *Config) validateCentralOverride(path *field.Path, config CentralOverride) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, c.validateInstanceIDs(path.Child("instanceIds"), config.InstanceIDs)...)
	errs = append(errs, c.validatePatch(path.Child("patch"), config.Patch)...)
	return errs
}

func (c *Config) validatePatch(path *field.Path, patch string) field.ErrorList {
	var errs field.ErrorList
	if len(patch) == 0 {
		errs = append(errs, field.Required(path, "patch is required"))
	}

	params := CentralParams{Name: "central-test-patch"}
	gitopsService := NewService(NewProviderWithReader(NewStaticReader(*c)))
	_, err := gitopsService.GetCentral(params)
	if err != nil {
		errs = append(errs, field.Invalid(path, patch, "invalid patch: "+err.Error()))
	}

	return errs
}

func (c *Config) validateInstanceIDs(path *field.Path, instanceIDs []string) field.ErrorList {
	var errs field.ErrorList
	var seenInstanceIDs = make(map[string]struct{})
	for i, instanceID := range instanceIDs {
		if _, ok := seenInstanceIDs[instanceID]; ok {
			errs = append(errs, field.Duplicate(path, instanceID))
		}
		seenInstanceIDs[instanceID] = struct{}{}
		errs = append(errs, c.validateInstanceID(path.Index(i), instanceID)...)
	}
	return errs
}

func (c *Config) validateInstanceID(path *field.Path, instanceID string) field.ErrorList {
	var errs field.ErrorList
	if len(instanceID) == 0 {
		errs = append(errs, field.Required(path, "instance ID is required"))
	}
	return errs
}
