// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace        = "stackrox-operator"
	releaseName              = "rhacs-operator"
	operatorDeploymentPrefix = "rhacs-operator-manager"

	// deployment names should contain at most 63 characters
	// RFC 1035 Label Names: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
	maxOperatorDeploymentNameLength = 63
)

func parseOperatorImages(images []ACSOperatorImage) ([]chartutil.Values, string, error) {
	var operatorImages []chartutil.Values
	var crdTag string
	uniqueImages := make(map[string]bool)
	for _, img := range images {
		if !strings.Contains(img.Image, ":") {
			return nil, "", fmt.Errorf("failed to parse image %q", img.Image)
		}
		strs := strings.Split(img.Image, ":")
		if len(strs) != 2 {
			return nil, "", fmt.Errorf("failed to split image and tag from %q", img.Image)
		}
		repo, tag := strs[0], strs[1]
		if len(operatorDeploymentPrefix+"-"+tag) > maxOperatorDeploymentNameLength {
			return nil, "", fmt.Errorf("%s-%s contains more than %d characters and cannot be used as a deployment name", operatorDeploymentPrefix, tag, maxOperatorDeploymentNameLength)
		}
		if img.InstallCRD {
			crdTag = tag
		}
		if _, used := uniqueImages[repo+tag]; !used {
			uniqueImages[repo+tag] = true
			img := chartutil.Values{"repository": repo, "tag": tag}
			operatorImages = append(operatorImages, img)
		}
	}
	return operatorImages, crdTag, nil
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	crdURL         string
	resourcesChart *chart.Chart
}

// ACSOperatorImage operator image representation which tells when to download CRD or skip it
type ACSOperatorImage struct {
	Image      string
	InstallCRD bool
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, images []ACSOperatorImage) error {
	if len(images) == 0 {
		return nil
	}

	operatorImages, crdTag, err := parseOperatorImages(images)
	if err != nil {
		return fmt.Errorf("failed to parse images: %w", err)
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"deploymentPrefix": operatorDeploymentPrefix + "-",
			"images":           operatorImages,
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

// DeleteOperator removes specified operator deployment from the cluster
func (u *ACSOperatorManager) DeleteOperator(ctx context.Context, version string) error {
	depName := operatorDeploymentPrefix + "-" + version
	dep := &appsv1.Deployment{}
	err := u.client.Get(ctx, ctrlClient.ObjectKey{Namespace: operatorNamespace, Name: depName}, dep)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("retrieving operator deployment version %s: %w", version, err)
	}
	err = u.client.Delete(ctx, dep)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("deleting operator deployment version %s: %w", version, err)
	}
	return nil
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
