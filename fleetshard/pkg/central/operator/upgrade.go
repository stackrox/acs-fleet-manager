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
	operatorNamespace = "stackrox-operator"
	releaseName       = "rhacs-operator"
	crdKind           = "CustomResourceDefinition"
)

func parseOperatorImages(images []string) ([]chartutil.Values, error) {
	var operatorImages []chartutil.Values
	for _, img := range images {
		if strings.Contains(img, ":") {
			s := strings.Split(img, ":")
			img := chartutil.Values{"repository": s[0], "tag": s[1]}
			operatorImages = append(operatorImages, img)
		} else {
			glog.Errorf("failed to parse image %s", img)
		}
	}
	if len(operatorImages) == 0 {
		return nil, fmt.Errorf("zero tags parsed from images %s", strings.Join(images, ", "))
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
			"images": operatorImages,
		},
	}

	u.resourcesChart = charts.MustGetChart("rhacs-operator")
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return fmt.Errorf("failed rendering operator chart: %w", err)
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" && obj.GetKind() != crdKind {
			obj.SetNamespace(operatorNamespace)
		}
		key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
		var out unstructured.Unstructured
		out.SetGroupVersionKind(obj.GroupVersionKind())
		err := u.client.Get(ctx, key, &out)
		if err == nil {
			glog.V(10).Infof("Updating %s/%s", obj.GetKind(), obj.GetName())
			obj.SetResourceVersion(out.GetResourceVersion())
			err := u.client.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("failed to update object %s/%s: %w", obj.GetKind(), key.Name, err)
			}
		} else {
			if !apiErrors.IsNotFound(err) {
				return fmt.Errorf("failed to retrieve object %s/%s: %w", obj.GetKind(), key.Name, err)
			}
			err = u.client.Create(ctx, obj)
			glog.Infof("Creating %s/%s", obj.GetKind(), obj.GetName())
			if err != nil && !apiErrors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create object %s/%s of type %s: %w", key.Namespace, key.Name, obj.GetKind(), err)
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
