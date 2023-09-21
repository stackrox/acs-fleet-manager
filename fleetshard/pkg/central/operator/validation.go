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
	for i, config := range configs {
		errs = append(errs, validateOperatorConfig(path.Index(i), config)...)
	}
	return errs
}

func validateOperatorConfig(path *field.Path, config OperatorConfig) field.ErrorList {
	var errs field.ErrorList

	errs = append(errs, validateDeploymentName(path.Child(keyDeploymentName), config[keyDeploymentName])...)
	errs = append(errs, validateImage(path.Child(keyImage), config[keyImage])...)
	errs = append(errs, validateDisableCentralReconciler(path.Child(keyDisableCentralReconciler), config[keyDisableCentralReconciler])...)
	errs = append(errs, validateDisableSecuredClusterReconciler(path.Child(keyDisableSecuredClusterReconciler), config[keyDisableSecuredClusterReconciler])...)
	errs = append(errs, validateLabelSelector(path.Child(keyCentralLabelSelector), config[keyCentralLabelSelector])...)
	errs = append(errs, validateLabelSelector(path.Child(keySecuredClusterSelector), config[keySecuredClusterSelector])...)
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

func validateDisableCentralReconciler(path *field.Path, disableCentralReconcilerIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	if disableCentralReconcilerIntf == nil {
		return nil
	}
	disableCentralReconciler, ok := disableCentralReconcilerIntf.(bool)
	if !ok {
		errs = append(errs, field.Invalid(path, disableCentralReconciler, "disableCentralReconciler must be a boolean"))
	}
	return errs
}

func validateDisableSecuredClusterReconciler(path *field.Path, disableSecuredClusterReconcilerIntf interface{}) field.ErrorList {
	var errs field.ErrorList
	if disableSecuredClusterReconcilerIntf == nil {
		return nil
	}
	disableSecuredClusterReconciler, ok := disableSecuredClusterReconcilerIntf.(bool)
	if !ok {
		errs = append(errs, field.Invalid(path, disableSecuredClusterReconciler, "disableSecuredClusterReconciler must be a boolean"))
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
