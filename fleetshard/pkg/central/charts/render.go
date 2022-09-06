package charts

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RenderToObjects renders the given release/chart, and returns the list of parsed (unstructured) objects contained
// in the chart.
func RenderToObjects(releaseName, namespace string, chrt *chart.Chart, values chartutil.Values) ([]*unstructured.Unstructured, error) {
	releaseOpts := chartutil.ReleaseOptions{
		Name:      releaseName,
		Namespace: namespace,
		IsUpgrade: true,
		Revision:  1,
	}
	renderVals, err := chartutil.ToRenderValues(chrt, values, releaseOpts, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create render values for chart %s: %w", chrt.Name(), err)
	}
	renderedFiles, err := engine.Render(chrt, renderVals)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart %s: %w", chrt.Name(), err)
	}

	var allObjs []*unstructured.Unstructured
	for fileName, contents := range renderedFiles {
		if !strings.HasSuffix(fileName, ".yaml") {
			continue
		}
		objs, err := k8sutil.UnstructuredFromYAMLMulti(contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file %s in chart %s: %w", fileName, chrt.Name(), err)
		}
		for _, obj := range objs {
			obj := obj
			allObjs = append(allObjs, &obj)
		}
	}

	return allObjs, nil
}
