// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"golang.org/x/exp/slices"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace        = "rhacs"
	releaseName              = "rhacs-operator"
	operatorDeploymentPrefix = "rhacs-operator"

	// deployment names should contain at most 63 characters
	// RFC 1035 Label Names: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
	maxOperatorDeploymentNameLength = 63
)

// GetRepoAndTagFromImage returns the repo and image tag
func GetRepoAndTagFromImage(img string) (string, string, error) {
	if !strings.Contains(img, ":") {
		return "", "", fmt.Errorf("failed to parse image %q", img)
	}
	strs := strings.Split(img, ":")
	if len(strs) != 2 {
		return "", "", fmt.Errorf("failed to split image and tag from %q", img)
	}
	repo, tag := strs[0], strs[1]

	return repo, tag, nil
}

func parseOperatorImages(images []string) ([]chartutil.Values, error) {
	var operatorImages []chartutil.Values
	uniqueImages := make(map[string]bool)
	for _, img := range images {
		repo, tag, err := GetRepoAndTagFromImage(img)
		if err != nil {
			return nil, err
		}

		deploymentName := generateDeploymentName(tag)
		if len(deploymentName) > maxOperatorDeploymentNameLength {
			return nil, fmt.Errorf("%s contains more than %d characters and cannot be used as a deployment name", deploymentName, maxOperatorDeploymentNameLength)
		}
		if _, used := uniqueImages[repo+tag]; !used {
			uniqueImages[repo+tag] = true
			img := chartutil.Values{
				"deploymentName": deploymentName,
				"repository":     repo,
				"tag":            tag,
				"labelSelector":  GetValidSelectorTag(tag),
			}
			operatorImages = append(operatorImages, img)
		}
	}
	return operatorImages, nil
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	crdURL         string
	resourcesChart *chart.Chart
}

var urlRegexExp *regexp.Regexp

func init() {
	var err error
	urlRegexExp, err = regexp.Compile("/[^a-z0-9\\-_]/g")
	if err != nil {
		panic(fmt.Errorf("invalid URL regex, could not be compiled %w", err))
	}
}

// GetValidSelectorTag returns a valid selector string which can be used in a kubernetes metadata label.
func GetValidSelectorTag(tag string) string {
	return urlRegexExp.ReplaceAllString(tag, "")
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, images []string, crdTag string) error {
	if len(images) == 0 {
		return nil
	}

	operatorImages, err := parseOperatorImages(images)
	if err != nil {
		return fmt.Errorf("failed to parse images: %w", err)
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"images":               operatorImages,
			"centralLabelSelector": true,
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

func generateDeploymentName(tag string) string {
	return operatorDeploymentPrefix + "-" + tag
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
