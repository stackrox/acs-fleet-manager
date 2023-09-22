package operator

import (
	"fmt"
	"net/url"

	"github.com/containers/image/docker/reference"
	"k8s.io/apimachinery/pkg/api/validation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate validates the operator configuration and can be used in different life-cycle stages like runtime and deploy time.
func Validate(path *field.Path, configs OperatorConfigs) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateCRDUrls(path.Child("crdUrls"), configs.CRDURLs)...)
	errs = append(errs, validateOperatorConfigs(path.Child("operators"), configs.Configs)...)
	if len(errs) == 0 {
		errs = append(errs, validateManifests(path, configs)...)
	}
	return errs
}

func validateManifests(path *field.Path, configs OperatorConfigs) field.ErrorList {
	var errs field.ErrorList
	manifests, err := RenderChart(configs)
	if err != nil {
		errs = append(errs, field.Invalid(path, nil, fmt.Sprintf("could not render operator helm charts: %s", err.Error())))
		return errs
	}
	if len(configs.Configs) > 0 && len(manifests) == 0 {
		errs = append(errs, field.Invalid(path, nil, "operator chart rendering succeed, but no manifests were rendered"))
		return errs
	}
	return nil
}

func validateCRDUrls(path *field.Path, urls []string) field.ErrorList {
	var errs field.ErrorList
	for i, urlStr := range urls {
		errs = append(errs, validateCRDURL(path.Index(i), urlStr)...)
	}
	return errs
}

func validateCRDURL(path *field.Path, urlStr string) field.ErrorList {
	var errs field.ErrorList
	if _, err := url.Parse(urlStr); err != nil {
		errs = append(errs, field.Invalid(path, urlStr, fmt.Sprintf("invalid url: %s", err.Error())))
	}
	return errs
}

func validateOperatorConfigs(path *field.Path, configs []OperatorConfig) field.ErrorList {
	var errs field.ErrorList
	seenDeploymentNames := make(map[string]struct{})
	for i, config := range configs {
		// Check that each deployment name is unique
		if _, ok := seenDeploymentNames[config.GetDeploymentName()]; ok {
			errs = append(errs, field.Duplicate(path.Index(i).Child("deploymentName"), config.GetDeploymentName()))
		}
		seenDeploymentNames[config.GetDeploymentName()] = struct{}{}
		errs = append(errs, validateOperatorConfig(path.Index(i), config)...)
	}
	return errs
}

func validateOperatorConfig(path *field.Path, config OperatorConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateDeploymentName(path.Child(keyDeploymentName), config[keyDeploymentName])...)
	errs = append(errs, validateImage(path.Child(keyImage), config[keyImage])...)
	errs = append(errs, validateCentralReconcilerEnabled(path.Child(keyCentralReconcilerEnabled), config[keyCentralReconcilerEnabled])...)
	errs = append(errs, validateSecuredClusterReconcilerEnabled(path.Child(keySecuredClusterReconcilerEnabled), config[keySecuredClusterReconcilerEnabled])...)
	errs = append(errs, validateLabelSelector(path.Child(keyCentralLabelSelector), config[keyCentralLabelSelector])...)
	errs = append(errs, validateLabelSelector(path.Child(keySecuredClusterSelector), config[keySecuredClusterSelector])...)
	errs = append(errs, validateHasCentralSelectorOrReconcilerDisabled(path, config)...)
	errs = append(errs, validateHasSecuredClusterSelectorOrReconcilerDisabled(path, config)...)
	return errs
}

func validateHasCentralSelectorOrReconcilerDisabled(path *field.Path, config OperatorConfig) field.ErrorList {
	var errs field.ErrorList
	if len(config.GetCentralLabelSelector()) == 0 && !config.GetCentralReconcilerEnabled() {
		errs = append(errs, field.Invalid(path, nil, "central label selector must be specified or central reconciler must be disabled"))
	}
	return errs
}

func validateHasSecuredClusterSelectorOrReconcilerDisabled(path *field.Path, config OperatorConfig) field.ErrorList {
	var errs field.ErrorList
	if len(config.GetSecuredClusterLabelSelector()) == 0 && !config.GetSecuredClusterReconcilerEnabled() {
		errs = append(errs, field.Invalid(path, nil, "secured cluster label selector must be specified or secured cluster reconciler must be disabled"))
	}
	return errs
}

func validateDeploymentName(path *field.Path, deploymentNameIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	deploymentName, ok := deploymentNameIntf.(string)
	if !ok {
		errs = append(errs, field.Invalid(path, deploymentName, "deployment name must be a string"))
	}
	if len(deploymentName) == 0 {
		errs = append(errs, field.Invalid(path, deploymentName, "deployment name cannot be empty"))
		return errs
	}
	if dnsErrs := validation.NameIsDNSSubdomain(deploymentName, true); len(dnsErrs) > 0 {
		errs = append(errs, field.Invalid(path, deploymentName, fmt.Sprintf("invalid deployment name: %v", dnsErrs[0])))
	}
	return errs
}

func validateImage(path *field.Path, imageIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	image, ok := imageIntf.(string)
	if !ok {
		errs = append(errs, field.Invalid(path, image, "image must be a string"))
		return errs
	}
	if len(image) == 0 {
		errs = append(errs, field.Invalid(path, image, "image cannot be empty"))
		return errs
	}
	_, err := reference.Parse(image)
	if err != nil {
		errs = append(errs, field.Invalid(path, image, fmt.Sprintf("invalid image: %v", err)))
		return errs
	}
	return errs
}

func validateCentralReconcilerEnabled(path *field.Path, centralReconcilerEnabledIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	if centralReconcilerEnabledIntf == nil {
		return nil
	}
	centralReconcilerEnabled, ok := centralReconcilerEnabledIntf.(bool)
	if !ok {
		errs = append(errs, field.Invalid(path, centralReconcilerEnabled, "centralReconcilerEnabled must be a boolean"))
	}
	return errs
}

func validateSecuredClusterReconcilerEnabled(path *field.Path, securedClusterReconcilerEnabledIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	if securedClusterReconcilerEnabledIntf == nil {
		return nil
	}
	securedClusterReconcilerEnabled, ok := securedClusterReconcilerEnabledIntf.(bool)
	if !ok {
		errs = append(errs, field.Invalid(path, securedClusterReconcilerEnabled, "securedClusterReconcilerEnabled must be a boolean"))
	}
	return errs
}

func validateLabelSelector(path *field.Path, selectorIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	if selectorIntf == nil {
		return nil
	}
	selectorStr, ok := selectorIntf.(string)
	if !ok {
		errs = append(errs, field.Invalid(path, selectorStr, "label selector must be a string"))
		return errs
	}
	_, err := v1.ParseToLabelSelector(selectorStr)
	if err != nil {
		errs = append(errs, field.Invalid(path, selectorStr, fmt.Sprintf("invalid label selector: %v", err)))
	}
	return errs
}
