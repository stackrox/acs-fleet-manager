// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	containerImage "github.com/containers/image/docker/reference"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"golang.org/x/exp/slices"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace        = "rhacs"
	releaseName              = "rhacs-operator"
	operatorDeploymentPrefix = "rhacs-operator"
)

// DeploymentConfig represents operator configuration for deployment
type DeploymentConfig struct {
	Image  string
	GitRef string
}

func parseOperatorConfigs(operators []DeploymentConfig) ([]chartutil.Values, error) {
	var helmValues []chartutil.Values
	for _, operator := range operators {
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
		helmValues = append(helmValues, operatorValues)
	}
	return helmValues, nil
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	crdURL         string
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, operators []DeploymentConfig, crdTag string) error {
	if len(operators) == 0 {
		return nil
	}

	operatorImages, err := parseOperatorConfigs(operators)
	if err != nil {
		return fmt.Errorf("failed to parse images: %w", err)
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"images": operatorImages,
		},
	}

	var dynamicTemplatesUrls []string
	if crdTag != "" {
		dynamicTemplatesUrls = u.generateCRDTemplateUrls(crdTag)
	}

	u.resourcesChart = charts.MustGetChart("rhacs-operator", dynamicTemplatesUrls)
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return fmt.Errorf("failed rendering operator chart: %w", err)
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(operatorNamespace)
		}
		err := charts.InstallOrUpdateChart(ctx, obj, u.client)
		if err != nil {
			return fmt.Errorf("failed to update operator object %w", err)
		}
	}

	return nil

}

// ListVersionsWithReplicas returns currently running ACS Operator versions with number of ready replicas
func (u *ACSOperatorManager) ListVersionsWithReplicas(ctx context.Context) (map[string]int32, error) {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := u.client.List(ctx, deployments,
		ctrlClient.InNamespace(operatorNamespace),
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
		ctrlClient.InNamespace(operatorNamespace),
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
		err := u.client.Get(ctx, ctrlClient.ObjectKey{Namespace: operatorNamespace, Name: deploymentName}, deployment)
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

func generateDeploymentName(version string) string {
	return operatorDeploymentPrefix + "-" + version
}

func (u *ACSOperatorManager) generateCRDTemplateUrls(tag string) []string {
	stackroxWithTag := fmt.Sprintf(u.crdURL, tag)
	centralCrdURL := stackroxWithTag + "platform.stackrox.io_centrals.yaml"
	securedClusterCrdURL := stackroxWithTag + "platform.stackrox.io_securedclusters.yaml"
	return []string{centralCrdURL, securedClusterCrdURL}
}

// NewACSOperatorManager creates a new ACS Operator Manager
func NewACSOperatorManager(k8sClient ctrlClient.Client, baseCrdURL string) *ACSOperatorManager {
	return &ACSOperatorManager{
		client: k8sClient,
		crdURL: baseCrdURL,
	}
}
