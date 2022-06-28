package centralreconciler

import (
	"context"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	centralName        = "test-central"
	conditionTypeReady = "Ready"
)

var simpleManagedCentral = private.ManagedCentral{
	Metadata: private.ManagedCentralAllOfMetadata{
		Name:      centralName,
		Namespace: centralName,
	},
}

func conditionForType(conditions []private.DataPlaneClusterUpdateStatusRequestConditions, conditionType string) (*private.DataPlaneClusterUpdateStatusRequestConditions, bool) {
	for _, c := range conditions {
		if c.Type == conditionType {
			return &c, true
		}
	}
	return nil, false
}

func TestReconcileCreate(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := CentralReconciler{
		status:    pointer.Int32(0),
		client:    fakeClient,
		central:   private.ManagedCentral{},
		useRoutes: true,
	}

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	if readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady); ok {
		assert.Equal(t, "True", readyCondition.Status)
	} else {
		assert.Fail(t, "Ready condition not found in conditions", status.Conditions)
	}

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralName}, central)
	require.NoError(t, err)
	assert.Equal(t, centralName, central.GetName())
	assert.Equal(t, "1", central.GetAnnotations()[revisionAnnotationKey])
	assert.Equal(t, true, *central.Spec.Central.Exposure.Route.Enabled)

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralName}, route)
	require.NoError(t, err)
	assert.Equal(t, centralReencryptRouteName, route.GetName())
	assert.Equal(t, openshiftRouteV1.TLSTerminationReencrypt, route.Spec.TLS.Termination)
	assert.Equal(t, testutils.CentralCA, route.Spec.TLS.DestinationCACertificate)
}

func TestReconcileUpdateSucceeds(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralName,
			Annotations: map[string]string{revisionAnnotationKey: "3"},
		},
	}).Build()

	r := CentralReconciler{
		status:  pointer.Int32(0),
		client:  fakeClient,
		central: private.ManagedCentral{},
	}

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	assert.Equal(t, "True", status.Conditions[0].Status)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralName}, central)
	require.NoError(t, err)
	assert.Equal(t, centralName, central.GetName())
	assert.Equal(t, "4", central.GetAnnotations()[revisionAnnotationKey])
}

func TestReconcileLastHashNotUpdatedOnError(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralName,
			Annotations: map[string]string{revisionAnnotationKey: "invalid annotation"},
		},
	}).Build()

	r := CentralReconciler{
		status:  pointer.Int32(0),
		client:  fakeClient,
		central: private.ManagedCentral{},
	}

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Error(t, err)

	assert.Equal(t, [16]byte{}, r.lastCentralHash)
}

func TestReconicleLastHashSetOnSuccess(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralName,
			Annotations: map[string]string{revisionAnnotationKey: "3"},
		},
	}).Build()

	r := CentralReconciler{
		status:  pointer.Int32(0),
		client:  fakeClient,
		central: private.ManagedCentral{},
	}

	expectedHash, err := util.MD5SumFromJSONStruct(&simpleManagedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	assert.Equal(t, expectedHash, r.lastCentralHash)

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	require.ErrorIs(t, err, ErrTypeCentralNotChanged)
}

func TestReconcileDelete(t *testing.T) {
	// given
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := CentralReconciler{
		status:    pointer.Int32(0),
		client:    fakeClient,
		central:   private.ManagedCentral{},
		useRoutes: true,
	}

	_, _ = r.Reconcile(context.TODO(), simpleManagedCentral)
	// when
	deletedCentral := simpleManagedCentral
	deletedCentral.Metadata.DeletionTimestamp = "2006-01-02T15:04:05Z07:00"

	maxReconcileAttempts := 10
	attempt := 0
	var status *private.DataPlaneCentralStatus
	var err error
	for attempt < maxReconcileAttempts && status == nil {
		status, err = r.Reconcile(context.TODO(), deletedCentral)
		attempt++
	}
	// then
	require.NoError(t, err)
	require.NotNil(t, status)

	if readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady); ok {
		assert.Equal(t, "False", readyCondition.Status)
		assert.Equal(t, "Deleted", readyCondition.Reason)
	} else {
		assert.Fail(t, "Ready condition not found in conditions", status.Conditions)
	}

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralName}, central)
	assert.True(t, errors.IsNotFound(err))

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralName}, route)
	assert.True(t, errors.IsNotFound(err))
}

func TestCentralChanged(t *testing.T) {

	tests := []struct {
		name           string
		lastCentral    *private.ManagedCentral
		currentCentral private.ManagedCentral
		want           bool
	}{
		{
			name:           "return true when lastCentral was not set",
			lastCentral:    nil,
			currentCentral: simpleManagedCentral,
			want:           true,
		},
		{
			name:           "return false when lastCentral equal currentCentral",
			lastCentral:    &simpleManagedCentral,
			currentCentral: simpleManagedCentral,
			want:           false,
		},
		{
			name:        "return true when lastCentral not equal currentCentral",
			lastCentral: &simpleManagedCentral,
			currentCentral: private.ManagedCentral{
				Metadata: simpleManagedCentral.Metadata,
				Spec: private.ManagedCentralAllOfSpec{
					Endpoint: private.ManagedCentralAllOfSpecEndpoint{
						Host: "central.cluster.local",
					},
				},
			},
			want: true,
		},
	}

	fakeClient := testutils.NewFakeClientBuilder(t).Build()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reconciler := CentralReconciler{
				status:  pointer.Int32(0),
				client:  fakeClient,
				central: test.currentCentral,
			}

			if test.lastCentral != nil {
				err := reconciler.setLastCentralHash(*test.lastCentral)
				require.NoError(t, err)
			}

			got, err := reconciler.centralChanged(test.currentCentral)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}

}
