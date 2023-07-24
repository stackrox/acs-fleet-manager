// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/utf8string"
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

	// RFC 1035 Label Names: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
	maxLabelLength = 63

	maxImageTagLength = 128
)

// DeploymentConfig represents operator configuration for deployment
type DeploymentConfig struct {
	Image         string
	LabelSelector string
	Version       string
}

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
	if !isValidTag(tag) {
		return "", "", fmt.Errorf("failed to validate image tag %q", tag)
	}

	return repo, tag, nil
}

// See image tag specification: https://docs.docker.com/engine/reference/commandline/tag/#description
func isValidTag(tag string) bool {
	if len(tag) == 0 || len(tag) > maxImageTagLength {
		return false
	}
	notAllowedStarts := []rune{'.', '-'}
	if slices.Contains(notAllowedStarts, rune(tag[0])) {
		return false
	}
	return utf8string.NewString(tag).IsASCII()
}

// IsValidLabel returns true if provided string could be a valid label
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func IsValidLabel(label string) bool {
	if len(label) > maxLabelLength {
		return false
	}
	if len(label) > 0 {
		notAllowedStartOrEnd := []rune{'-', '_', '.'}
		if slices.Contains(notAllowedStartOrEnd, rune(label[0])) {
			return false
		}
		if slices.Contains(notAllowedStartOrEnd, rune(label[len(label)-1])) {
			return false
		}
	}
	return utf8string.NewString(label).IsASCII()
}

func parseOperatorConfigs(operators []DeploymentConfig) ([]chartutil.Values, error) {
	var operatorImages []chartutil.Values
	uniqueImages := make(map[string]bool)
	for _, operator := range operators {
		repo, tag, err := GetRepoAndTagFromImage(operator.Image)
		if err != nil {
			return nil, err
		}

		deploymentName := generateDeploymentName(tag)
		// deployment name has the same requirements as label values
		if !IsValidLabel(deploymentName) {
			return nil, fmt.Errorf("deployment name %s is not valid", deploymentName)
		}
		if !IsValidLabel(operator.LabelSelector) {
			return nil, fmt.Errorf("label selector %s is not valid", operator.LabelSelector)
		}
		if _, used := uniqueImages[repo+tag]; !used {
			uniqueImages[repo+tag] = true
			img := chartutil.Values{
				"deploymentName": deploymentName,
				"repository":     repo,
				"tag":            tag,
				"labelSelector":  operator.LabelSelector,
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
