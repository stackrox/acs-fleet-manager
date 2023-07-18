package config

import (
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"gopkg.in/yaml.v2"
	field "k8s.io/apimachinery/pkg/util/validation/field"
)

// GitOpsConfig represents the declarative configuration for Central instances defaults,
// rollout groups and overrides.
type GitOpsConfig struct {
	// Default configuration for Central instances.
	Default GitOpsDefaultConfig `json:"default"`
	// Overrides are the overrides for Central instances.
	Overrides []GitOpsInstanceOverride `json:"overrides"`
}

// GitOpsDefaultConfig represents the default configuration for Central instances.
type GitOpsDefaultConfig struct {
	// DefaultCentral is the default Central instance configuration.
	DefaultCentral v1alpha1.Central `json:"central"`
	// DefaultOperatorVersion is the default operator version.
	OperatorVersion string `json:"operatorVersion"`
}

// GitOpsInstanceOverride represents the configuration for a Central instance override. The override
// will be applied on top of the default central instance configuration.
type GitOpsInstanceOverride struct {
	// InstanceID is the instance ID for which the override is applicable.
	InstanceID string `json:"instanceId"`
	// Patch is the patch for the override, which will be applied as a strategic merge patch.
	Patch string `json:"patch"`
}

// ValidateGitOpsConfig validates the GitOps configuration.
func ValidateGitOpsConfig(config GitOpsConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateGitOpsDefaultConfig(field.NewPath("default"), config.Default)...)
	errs = append(errs, validateGitOpsOverrides(field.NewPath("overrides"), config.Overrides)...)
	return errs
}

func validateGitOpsDefaultConfig(path *field.Path, config GitOpsDefaultConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateOperatorVersion(path.Child("operatorVersion"), config.OperatorVersion)...)
	return errs
}

func validateGitOpsOverrides(path *field.Path, config []GitOpsInstanceOverride) field.ErrorList {
	var errs field.ErrorList
	var seenInstanceIDs = make(map[string]struct{})
	for i, override := range config {
		if _, ok := seenInstanceIDs[override.InstanceID]; ok {
			errs = append(errs, field.Duplicate(path.Index(i).Child("instanceId"), override.InstanceID))
		}
		seenInstanceIDs[override.InstanceID] = struct{}{}
		errs = append(errs, validateGitOpsInstanceOverride(path.Index(i), override)...)
	}
	return errs
}

func validateGitOpsInstanceOverride(path *field.Path, config GitOpsInstanceOverride) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateInstanceID(path.Child("instanceId"), config.InstanceID)...)
	errs = append(errs, validatePatch(path.Child("patch"), config.Patch)...)
	return errs
}

func validateOperatorVersion(path *field.Path, operatorVersion string) field.ErrorList {
	var errs field.ErrorList
	if len(operatorVersion) == 0 {
		errs = append(errs, field.Required(path, "operator version is required"))
	}
	return errs
}

func validatePatch(path *field.Path, patch string) field.ErrorList {
	var errs field.ErrorList
	if len(patch) == 0 {
		errs = append(errs, field.Required(path, "patch is required"))
	}
	// try to unmarshal the patch into a Central instance to validate it
	if err := yaml.Unmarshal([]byte(patch), &v1alpha1.Central{}); err != nil {
		errs = append(errs, field.Invalid(path, patch, "invalid patch: "+err.Error()))
		return errs
	}
	return errs
}

func validateInstanceID(path *field.Path, instanceID string) field.ErrorList {
	var errs field.ErrorList
	if len(instanceID) == 0 {
		errs = append(errs, field.Required(path, "instance ID is required"))
	}
	return errs
}
