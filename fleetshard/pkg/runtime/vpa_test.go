package runtime

import (
	"context"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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

var fakeRestMapper meta.RESTMapper = &fakeRESTMapper{scopeForGvk: map[schema.GroupVersionKind]meta.RESTScope{
	{Group: "apps", Version: "v1", Kind: "Deployment"}:                              meta.RESTScopeNamespace,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:                              meta.RESTScopeNamespace,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}:        meta.RESTScopeRoot,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}: meta.RESTScopeRoot,
}}

func Test_vpaReconciler_Reconcile(t *testing.T) {
	cli := fake.NewFakeClient()
	v := newVPAReconciler(cli, fakeRestMapper)
	err := v.reconcile(context.Background(), private.VerticalPodAutoscaling{
		Recommenders: []private.VpaRecommenderConfig{
			{
				Name:  "recommender-1",
				Image: "image-1",
				Resources: private.ResourceRequirements{
					Requests: map[string]string{
						"cpu":    "100m",
						"memory": "100Mi",
					},
					Limits: map[string]string{
						"cpu":    "100m",
						"memory": "100Mi",
					},
				},
			},
		},
	})

	require.NoError(t, err)

	var deployments appsv1.DeploymentList
	err = cli.List(context.Background(), &deployments, client.InNamespace("rhacs-vertical-pod-autoscaling"))
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 1)
	assert.Equal(t, "recommender-1", deployments.Items[0].Name)
	require.Len(t, deployments.Items[0].Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "image-1", deployments.Items[0].Spec.Template.Spec.Containers[0].Image)

	var sa v1.ServiceAccount
	err = cli.Get(context.Background(), client.ObjectKey{Namespace: "rhacs-vertical-pod-autoscaling", Name: "rhacs-vpa-recommender"}, &sa)
	require.NoError(t, err)

	var clusterRole rbacv1.ClusterRole
	err = cli.Get(context.Background(), client.ObjectKey{Name: "rhacs-vpa-recommender"}, &clusterRole)
	require.NoError(t, err)

	var clusterRoleBinding rbacv1.ClusterRoleBinding
	err = cli.Get(context.Background(), client.ObjectKey{Name: "rhacs-vpa-recommender"}, &clusterRoleBinding)
	require.NoError(t, err)

}
