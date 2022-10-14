package k8s

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type restScope struct{}

func (r *restScope) Name() meta.RESTScopeName {
	return "namespace/name"
}

func TestIsRoutesResourceEnabled(t *testing.T) {
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{routesGVK.GroupVersion()})
	mapper.Add(schema.GroupVersionKind{Group: routesGVK.Group, Version: routesGVK.Version, Kind: "Route"}, &restScope{})

	fakeClient := testutils.NewFakeClientBuilder(t).
		WithRESTMapper(mapper).
		Build()
	enabled, err := IsRoutesResourceEnabled(fakeClient)
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestIsRoutesResourceEnabledShouldReturnFalse(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	enabled, err := IsRoutesResourceEnabled(fakeClient)
	require.NoError(t, err)
	assert.False(t, enabled)
}
