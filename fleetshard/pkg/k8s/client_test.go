package k8s

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestIsRoutesResourceEnabled(t *testing.T) {
	//fakeClient := testutils.NewFakeClientBuilder(t).Build()
	fakeClient := CreateClientOrDie()
	enabled, err := IsRoutesResourceEnabled(context.TODO(), fakeClient)
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestIsRoutesResourceEnabledShouldReturnFalse(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	enabled, err := IsRoutesResourceEnabled(context.TODO(), fakeClient)
	require.NoError(t, err)
	assert.False(t, enabled)
}
