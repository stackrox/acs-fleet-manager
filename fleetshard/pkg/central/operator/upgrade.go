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

// chartValue represents operator image
type chartValue struct {
	repository string
	tags       []string
}

func parseRepositoryAndTags(images []string) (*chartValue, error) {
	val := chartValue{}
	for _, img := range images {
		if strings.Contains(img, ":") {
			s := strings.Split(img, ":")
			val.repository = s[0]
			val.tags = append(val.tags, s[1])
		} else {
			glog.Errorf("failed to parse image %s", img)
		}
	}
	if len(val.tags) == 0 {
		return nil, fmt.Errorf("zero tags parsed from images %s", strings.Join(images, ", "))
	}

	return &val, nil
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, images []string) error {
	val, err := parseRepositoryAndTags(images)
	if err != nil {
		return fmt.Errorf("failed to parse images: %w", err)
	}
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"repository": val.repository,
			"tags":       val.tags,
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
			glog.V(10).Infof("Updating object %s/%s", obj.GetNamespace(), obj.GetName())
			obj.SetResourceVersion(out.GetResourceVersion())
			err := u.client.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("failed to update object %s/%s of type %s %w", key.Namespace, key.Name, obj.GetKind(), err)
			}
		} else {
			if !apiErrors.IsNotFound(err) {
				return fmt.Errorf("failed to retrieve object %s/%s of type %s %w", key.Namespace, key.Name, obj.GetKind(), err)
			}
			err = u.client.Create(ctx, obj)
			glog.Infof("Creating object %s/%s", obj.GetNamespace(), obj.GetName())
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
