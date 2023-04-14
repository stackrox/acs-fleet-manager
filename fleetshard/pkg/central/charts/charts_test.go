package charts

import (
	"context"
	"embed"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
)

//go:embed testdata
var testdata embed.FS

var testNamespace = "test-namespace"

var dummyDeployment = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "dummy",
		Namespace: testNamespace,
		Labels: map[string]string{
			"foo": "bar",
		},
	},
}

func TestTenantResourcesChart(t *testing.T) {
	c, err := GetChart("tenant-resources")
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestInstallOrUpdateChartCreateNew(t *testing.T) {
	chart, err := LoadChart(testdata, "testdata/test-chart")
	require.NoError(t, err)
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	ctx := context.Background()

	chartVals := chartutil.Values{
		"foo": "bar",
	}
	objs, err := RenderToObjects("test-release", testNamespace, chart, chartVals)
	require.NoError(t, err)
	obj := objs[0]

	err = InstallOrUpdateChart(ctx, obj, fakeClient)
	require.NoError(t, err)

	key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	var res unstructured.Unstructured
	res.SetGroupVersionKind(obj.GroupVersionKind())

	err = fakeClient.Get(ctx, key, &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.GetLabels())
	assert.Equal(t, "bar", res.GetLabels()["foo"])
}

func TestInstallOrUpdateChartUpdateExisting(t *testing.T) {
	chart, err := LoadChart(testdata, "testdata/test-chart")
	require.NoError(t, err)
	fakeClient := testutils.NewFakeClientBuilder(t, dummyDeployment).Build()
	ctx := context.Background()

	chartVals := chartutil.Values{
		"foo": "baz",
	}
	objs, err := RenderToObjects("test-release", testNamespace, chart, chartVals)
	require.NoError(t, err)
	obj := objs[0]

	err = InstallOrUpdateChart(ctx, obj, fakeClient)
	require.NoError(t, err)

	key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	var res unstructured.Unstructured
	res.SetGroupVersionKind(obj.GroupVersionKind())

	err = fakeClient.Get(ctx, key, &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.GetLabels())
	assert.Equal(t, "baz", res.GetLabels()["foo"])
}
