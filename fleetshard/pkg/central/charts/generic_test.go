package charts

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

type fakeRESTMapper struct {
	meta.RESTMapper
	scopeForGvk map[schema.GroupVersionKind]meta.RESTScope
}

func (f *fakeRESTMapper) setMappingForGvk(gvk schema.GroupVersionKind, mapping *meta.RESTMapping) {
	f.scopeForGvk[gvk] = mapping.Scope
}

func (f *fakeRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	scope, ok := f.scopeForGvk[schema.GroupVersionKind{Group: gk.Group, Version: versions[0], Kind: gk.Kind}]
	if !ok {
		return nil, fmt.Errorf("no mapping found for %s", gk.String())
	}
	return &meta.RESTMapping{Scope: scope}, nil
}

var rm = &fakeRESTMapper{scopeForGvk: map[schema.GroupVersionKind]meta.RESTScope{
	{Group: "apps", Version: "v1", Kind: "Deployment"}:                       meta.RESTScopeNamespace,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:                       meta.RESTScopeNamespace,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}: meta.RESTScopeRoot,
}}

func getParams(t *testing.T, cli client.Client) HelmReconcilerParams {
	chart := mustGetChart(t, "test-chart-2")
	return HelmReconcilerParams{
		ReleaseName: "my-release",
		Namespace:   "my-namespace",
		ManagerName: "test",
		Chart:       chart,
		Values:      chartutil.Values{},
		Client:      cli,
		RestMapper:  rm,
		AllowedGVKs: []schema.GroupVersionKind{
			{Group: "apps", Version: "v1", Kind: "Deployment"},
			{Group: "", Version: "v1", Kind: "ServiceAccount"},
			{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		},
	}
}

func TestReconcile_ShouldCreateNamespacedResources(t *testing.T) {
	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	params.CreateNamespace = true
	err := Reconcile(context.Background(), params)
	require.NoError(t, err)

	var sa v1.ServiceAccount
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "my-namespace", Name: "dummy"}, &sa)
	require.NoError(t, err)

}

func TestReconcile_ShouldDeleteUnwantedNamespacedResources(t *testing.T) {
	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	params.CreateNamespace = true
	params.Values["enabled"] = true

	err := Reconcile(context.Background(), params)
	require.NoError(t, err)

	var deployment appsv1.Deployment
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: params.Namespace, Name: "dummy"}, &deployment)
	require.NoError(t, err)

	params.Values["enabled"] = false
	err = Reconcile(context.Background(), params)
	require.NoError(t, err)

	err = cli.Get(context.Background(), client.ObjectKey{Namespace: params.Namespace, Name: "dummy"}, &deployment)
	require.Error(t, err)
	require.Nil(t, client.IgnoreNotFound(err))

}

func TestReconcile_ShouldDeleteUnwantedClusterResources(t *testing.T) {
	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	params.CreateNamespace = true
	params.Values["enabled"] = true

	err := Reconcile(context.Background(), params)
	require.NoError(t, err)

	var clusterRole rbacv1.ClusterRole
	err = cli.Get(context.Background(), client.ObjectKey{Name: "dummy"}, &clusterRole)
	require.NoError(t, err)

	params.Values["enabled"] = false
	err = Reconcile(context.Background(), params)
	require.NoError(t, err)

	err = cli.Get(context.Background(), client.ObjectKey{Name: "dummy"}, &clusterRole)
	require.Error(t, err)
	require.Nil(t, client.IgnoreNotFound(err))

}

func TestReconcile_ShouldThrowIfUnregisteredGVK(t *testing.T) {
	// The allowed GVK is not present in the params.
	// The test-Chart-2 has a "Role" resource that is created
	// when .Values.forbidden = true

	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	params.Values["forbidden"] = true

	err := Reconcile(context.Background(), params)
	require.Error(t, err)

}

func TestReconcile_ShouldNotCreateNamespaceByDefault(t *testing.T) {
	// The allowed GVK is not present in the params.
	// The test-Chart-2 has a "Role" resource that is created
	// when .Values.forbidden = true

	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	err := Reconcile(context.Background(), params)
	require.NoError(t, err)

	var ns v1.Namespace
	err = cli.Get(context.Background(), client.ObjectKey{Name: params.Namespace}, &ns)
	require.Error(t, err)

}

func TestReconcile_ShouldCreateNamespace(t *testing.T) {
	// The allowed GVK is not present in the params.
	// The test-Chart-2 has a "Role" resource that is created
	// when .Values.forbidden = true

	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	params.CreateNamespace = true
	err := Reconcile(context.Background(), params)
	require.NoError(t, err)

	var ns v1.Namespace
	err = cli.Get(context.Background(), client.ObjectKey{Name: params.Namespace}, &ns)
	require.NoError(t, err)

}

func TestReconcile_ShouldFailIfNamespaceDeleting(t *testing.T) {
	// The allowed GVK is not present in the params.
	// The test-Chart-2 has a "Role" resource that is created
	// when .Values.forbidden = true

	cli := fake.NewFakeClient(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-namespace",
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Add(-1 * time.Hour),
			},
			Finalizers: []string{"kubernetes"},
		},
	})

	params := getParams(t, cli)
	params.CreateNamespace = true
	require.Error(t, Reconcile(context.Background(), params))

}

func TestReconcile_ShouldApplyOwnershipLabels(t *testing.T) {
	cli := fake.NewFakeClient()

	params := getParams(t, cli)
	params.CreateNamespace = true
	err := Reconcile(context.Background(), params)
	require.NoError(t, err)

	var sa v1.ServiceAccount
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: params.Namespace, Name: "dummy"}, &sa)
	require.NoError(t, err)
	assert.Equal(t, params.ReleaseName, sa.Labels[labelHelmReleaseName])
	assert.Equal(t, "test-resource-0.0.0", sa.Labels[labelHelmChart])
	assert.Equal(t, params.Namespace, sa.Labels[labelHelmReleaseNamespace])
	assert.Equal(t, params.ManagerName, sa.Labels[labelManagedBy])
}

func TestReconcile_ShouldFailIfManagedResourceExist(t *testing.T) {
	cli := fake.NewFakeClient(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "my-namespace",
			Labels: map[string]string{
				"helm.sh/release": "other",
			},
		},
	})

	params := getParams(t, cli)
	require.Error(t, Reconcile(context.Background(), params))
}

func TestReconcile_ShouldFailIfParamsInvalid(t *testing.T) {
	cli := fake.NewFakeClient()
	params := getParams(t, cli)
	params.Client = nil
	require.Error(t, Reconcile(context.Background(), params))
}

func TestValidateParams(t *testing.T) {

	cli := fake.NewFakeClient()

	tests := []struct {
		name   string
		params func() HelmReconcilerParams
	}{
		{
			name: "ReleaseName cannot be empty",
			params: func() HelmReconcilerParams {
				p := getParams(t, cli)
				p.ReleaseName = ""
				return p
			},
		}, {
			name: "Namespace cannot be empty",
			params: func() HelmReconcilerParams {
				p := getParams(t, cli)
				p.Namespace = ""
				return p
			},
		}, {
			name: "ManagerName cannot be empty",
			params: func() HelmReconcilerParams {
				p := getParams(t, cli)
				p.ManagerName = ""
				return p
			},
		}, {
			name: "Chart cannot be nil",
			params: func() HelmReconcilerParams {
				p := getParams(t, cli)
				p.Chart = nil
				return p
			},
		}, {
			name: "Client cannot be nil",
			params: func() HelmReconcilerParams {
				p := getParams(t, nil)
				return p
			},
		},
		{
			name: "AllowedGVKs cannot be empty",
			params: func() HelmReconcilerParams {
				p := getParams(t, cli)
				p.AllowedGVKs = nil
				return p
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParams(tt.params())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.name)
		})
	}
}
