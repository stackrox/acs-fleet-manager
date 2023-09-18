// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	containerImage "github.com/containers/image/docker/reference"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"html/template"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	// ACSOperatorNamespace default Operator Namespace
	ACSOperatorNamespace = "rhacs"
	// ACSOperatorConfigMap name for configMap with operator deployment configurations
	ACSOperatorConfigMap      = "operator-config"
	releaseName               = "rhacs-operator"
	operatorDeploymentPrefix  = "rhacs-operator"
	defaultCRDBaseURLTemplate = "https://raw.githubusercontent.com/stackrox/stackrox/{{ .GitRef }}/operator/bundle/manifests/"
)

func parseOperatorConfigs(operators OperatorConfigs) ([]chartutil.Values, error) {
	var helmValues []chartutil.Values
	for _, operator := range operators.Configs {
		imageReference, err := containerImage.Parse(operator.Image)
		if err != nil {
			return nil, err
		}
		image := imageReference.String()
		if errs := validation.IsValidLabelValue(operator.GitRef); errs != nil {
			return nil, fmt.Errorf("label selector %s is not valid: %v", operator.GitRef, errs)
		}

		deploymentName := generateDeploymentName(operator.GitRef)
		// validate deploymentName (RFC-1123)
		// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
		if errs := apimachineryvalidation.NameIsDNSSubdomain(deploymentName, true); errs != nil {
			return nil, fmt.Errorf("invalid deploymentName %s: %v", deploymentName, errs)
		}
		operatorValues := chartutil.Values{
			"deploymentName": deploymentName,
			"image":          image,
			"labelSelector":  operator.GitRef,
		}

		operatorHelmValues := make(map[string]interface{})
		err = yaml.Unmarshal([]byte(operator.HelmValues), operatorHelmValues)
		if err != nil {
			return nil, fmt.Errorf("Unmarshalling Helm values failed for operator %s: %w.", operator.GitRef, err)
		}

		chartutil.CoalesceTables(operatorValues, operatorHelmValues)
		helmValues = append(helmValues, operatorValues)
	}
	return helmValues, nil
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, operators OperatorConfigs) error {
	objs, err := u.RenderChart(operators)
	if err != nil {
		return err
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(ACSOperatorNamespace)
		}
		err := charts.InstallOrUpdateChart(ctx, obj, u.client)
		if err != nil {
			return fmt.Errorf("failed to update operator object %w", err)
		}
	}

	return nil

}

// RenderChart renders the operator helm chart manifests
func (u *ACSOperatorManager) RenderChart(operators OperatorConfigs) ([]*unstructured.Unstructured, error) {
	if len(operators.Configs) == 0 {
		return nil, nil
	}

	operatorImages, err := parseOperatorConfigs(operators)
	if err != nil {
		return nil, fmt.Errorf("failed to parse images: %w", err)
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"images": operatorImages,
		},
	}

	var dynamicTemplatesUrls []string
	if operators.CRD.GitRef != "" {
		dynamicTemplatesUrls, err = u.generateCRDTemplateUrls(operators.CRD)
		if err != nil {
			return nil, err
		}
	}

	u.resourcesChart, err = charts.GetChart("rhacs-operator", dynamicTemplatesUrls)
	if err != nil {
		return nil, fmt.Errorf("failed getting chart: %w", err)
	}
	objs, err := charts.RenderToObjects(releaseName, ACSOperatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return nil, fmt.Errorf("failed rendering operator chart: %w", err)
	}
	return objs, nil
}

// ListVersionsWithReplicas returns currently running ACS Operator versions with number of ready replicas
func (u *ACSOperatorManager) ListVersionsWithReplicas(ctx context.Context) (map[string]int32, error) {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := u.client.List(ctx, deployments,
		ctrlClient.InNamespace(ACSOperatorNamespace),
		ctrlClient.MatchingLabels(labels),
	)
	if err != nil {
		return nil, fmt.Errorf("failed list operator deployments: %w", err)
	}

	versionWithReplicas := make(map[string]int32)
	for _, dep := range deployments.Items {
		for _, c := range dep.Spec.Template.Spec.Containers {
			if c.Name == "manager" {
				versionWithReplicas[c.Image] = dep.Status.ReadyReplicas
			}
		}
	}

	return versionWithReplicas, nil
}

// RemoveUnusedOperators removes unused operator deployments from the cluster. It receives a list of operator images which should be present in the cluster and removes all deployments which do not deploy any of the desired images.
func (u *ACSOperatorManager) RemoveUnusedOperators(ctx context.Context, desiredImages []string) error {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := u.client.List(ctx, deployments,
		ctrlClient.InNamespace(ACSOperatorNamespace),
		ctrlClient.MatchingLabels(labels),
	)
	if err != nil {
		return fmt.Errorf("failed list operator deployments: %w", err)
	}

	var unusedDeployments []string
	for _, deployment := range deployments.Items {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == "manager" && !slices.Contains(desiredImages, container.Image) {
				unusedDeployments = append(unusedDeployments, deployment.Name)
			}
		}
	}

	for _, deploymentName := range unusedDeployments {
		deployment := &appsv1.Deployment{}
		err := u.client.Get(ctx, ctrlClient.ObjectKey{Namespace: ACSOperatorNamespace, Name: deploymentName}, deployment)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("retrieving operator deployment %s: %w", deploymentName, err)
		}
		err = u.client.Delete(ctx, deployment)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("deleting operator deployment %s: %w", deploymentName, err)
		}
	}

	return nil
}

// ReadOperatorConfigFromConfigMap reads Operator deployment configuration from ConfigMap
func (u *ACSOperatorManager) ReadOperatorConfigFromConfigMap(ctx context.Context) ([]OperatorConfig, error) {
	configMap := &v1.ConfigMap{}

	err := u.client.Get(ctx, ctrlClient.ObjectKey{Name: ACSOperatorConfigMap, Namespace: "acsms"}, configMap)
	if err != nil {
		return nil, fmt.Errorf("retrieving operators configMap: %v", err)
	}

	operatorsConfigYAML := configMap.Data["operator-config.yaml"]
	var configMapOperators []OperatorConfig

	err = yaml.Unmarshal([]byte(operatorsConfigYAML), &configMapOperators)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling operators configMap: %v", err)
	}

	return configMapOperators, nil
}

func generateDeploymentName(version string) string {
	return operatorDeploymentPrefix + "-" + version
}

func (u *ACSOperatorManager) generateCRDTemplateUrls(crdConfig CRDConfig) ([]string, error) {
	baseURL := defaultCRDBaseURLTemplate
	if crdConfig.BaseURL != "" {
		baseURL = crdConfig.BaseURL
	}

	wr := new(strings.Builder)
	crdURLTpl, err := template.New("crd_base_url").Parse(baseURL)
	if err != nil {
		return []string{}, fmt.Errorf("could not parse CRD base URL: %w", err)
	}

	err = crdURLTpl.Execute(wr, struct {
		GitRef string
	}{
		GitRef: crdConfig.GitRef,
	})
	if err != nil {
		return []string{}, fmt.Errorf("could not parse CRD base URL: %w", err)
	}

	//stackroxWithTag := fmt.Sprintf(baseURL, crdConfig.GitRef)
	centralCrdURL := wr.String() + "platform.stackrox.io_centrals.yaml"
	securedClusterCrdURL := wr.String() + "platform.stackrox.io_securedclusters.yaml"
	return []string{centralCrdURL, securedClusterCrdURL}, nil
}

// NewACSOperatorManager creates a new ACS Operator Manager
func NewACSOperatorManager(k8sClient ctrlClient.Client) *ACSOperatorManager {
	return &ACSOperatorManager{
		client: k8sClient,
	}
}
