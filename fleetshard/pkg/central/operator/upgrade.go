// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace        = "stackrox-operator"
	releaseName              = "rhacs-operator"
	operatorDeploymentPrefix = "rhacs-operator-manager"
	centralCrdURL            = "https://raw.githubusercontent.com/stackrox/stackrox/master/operator/bundle/manifests/platform.stackrox.io_centrals.yaml"
	securedClusterCrdURL     = "https://raw.githubusercontent.com/stackrox/stackrox/master/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml"

	// deployment names should contain at most 63 characters
	// RFC 1035 Label Names: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
	maxOperatorDeploymentNameLength = 63
)

func parseOperatorImages(images []string) ([]chartutil.Values, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("the list of images is empty")
	}
	var operatorImages []chartutil.Values
	uniqueImages := make(map[string]bool)
	for _, img := range images {
		if !strings.Contains(img, ":") {
			return nil, fmt.Errorf("failed to parse image %q", img)
		}
		strs := strings.Split(img, ":")
		if len(strs) != 2 {
			return nil, fmt.Errorf("failed to split image and tag from %q", img)
		}
		repo, tag := strs[0], strs[1]
		if len(operatorDeploymentPrefix+"-"+tag) > maxOperatorDeploymentNameLength {
			return nil, fmt.Errorf("%s-%s contains more than %d characters and cannot be used as a deployment name", operatorDeploymentPrefix, tag, maxOperatorDeploymentNameLength)
		}
		if _, used := uniqueImages[repo+tag]; !used {
			uniqueImages[repo+tag] = true
			img := chartutil.Values{"repository": repo, "tag": tag}
			operatorImages = append(operatorImages, img)
		}
	}
	return operatorImages, nil
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, images []string) error {
	operatorImages, err := parseOperatorImages(images)
	if err != nil {
		return fmt.Errorf("failed to parse images: %w", err)
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"deploymentPrefix": operatorDeploymentPrefix + "-",
			"images":           operatorImages,
		},
	}

	if err := charts.DownloadTemplate(centralCrdURL, "rhacs-operator"); err != nil {
		return fmt.Errorf("failed to download %s: %w", centralCrdURL, err)
	}
	if err := charts.DownloadTemplate(securedClusterCrdURL, "rhacs-operator"); err != nil {
		return fmt.Errorf("failed to download %s: %w", securedClusterCrdURL, err)
	}
	u.resourcesChart = charts.MustGetChart("rhacs-operator")
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return fmt.Errorf("failed rendering operator chart: %w", err)
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(operatorNamespace)
		}
		key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
		var out unstructured.Unstructured
		out.SetGroupVersionKind(obj.GroupVersionKind())
		err := u.client.Get(ctx, key, &out)
		if err == nil {
			glog.V(10).Infof("Updating %s/%s in %s namespace", obj.GetKind(), obj.GetName(), obj.GetNamespace())
			obj.SetResourceVersion(out.GetResourceVersion())
			err := u.client.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("failed to update object %s/%s in %s namespace: %w", obj.GetKind(), key.Name, key.Namespace, err)
			}
		} else {
			if !apiErrors.IsNotFound(err) {
				return fmt.Errorf("failed to retrieve object %s/%s in %s namespace: %w", obj.GetKind(), key.Name, key.Namespace, err)
			}
			err = u.client.Create(ctx, obj)
			glog.Infof("Creating %s/%s in %s namespace", obj.GetKind(), obj.GetName(), obj.GetNamespace())
			if err != nil && !apiErrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create object %s/%s in %s namespace: %w", obj.GetKind(), key.Name, key.Namespace, err)
			}
		}
	}

	return nil

}

// NewACSOperatorManager creates a new ACS Operator Manager
func NewACSOperatorManager(k8sClient ctrlClient.Client) *ACSOperatorManager {
	return &ACSOperatorManager{
		client: k8sClient,
	}
}
