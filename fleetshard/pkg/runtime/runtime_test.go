package runtime

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/require"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func TestRuntime(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	fleetManagerClientMock := &fleetmanager.FleetManagerClientMock{}
	fleetManagerClientMock.GetManagedCentralListFunc = func() (*private.ManagedCentralList, error) {
		return &private.ManagedCentralList{
			Items: []private.ManagedCentral{
				{
					Id:   "id",
					Kind: "",
					Spec: private.ManagedCentralAllOfSpec{},
					Metadata: private.ManagedCentralAllOfMetadata{
						Name:      "test-central",
						Namespace: "test-namespace",
					},
				},
			},
		}, nil
	}
	fleetManagerClientMock.UpdateStatusFunc = func(statuses map[string]private.DataPlaneCentralStatus) error {
		return nil
	}

	runtime := Runtime{
		k8sClient: fakeClient,
		client:    fleetManagerClientMock,
		config: &config.Config{
			CreateAuthProvider: false,
			RuntimePollPeriod:  1 * time.Second,
		},
		reconcilers: reconcilerRegistry{},
	}

	err := runtime.Start()
	require.NoError(t, err)

	time.Sleep(2 * time.Second)
	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: "test-central", Namespace: "test-namespace"}, central)
	require.NoError(t, err)
	//assert.Len(t, fleetManagerClientMock.UpdateStatusCalls(), 1)

	fleetManagerClientMock.GetManagedCentralListFunc = func() (*private.ManagedCentralList, error) {
		return &private.ManagedCentralList{
			Items: []private.ManagedCentral{
				{
					Id:   "id",
					Kind: "",
					Spec: private.ManagedCentralAllOfSpec{},
					Metadata: private.ManagedCentralAllOfMetadata{
						Name:              "test-central",
						Namespace:         "test-namespace",
						DeletionTimestamp: time.Now().String(),
					},
				},
			},
		}, nil
	}

	time.Sleep(5 * time.Second)
	err = fakeClient.Get(context.TODO(), ctrlClient.ObjectKey{Name: "test-central", Namespace: "test-namespace"}, central)
	require.True(t, apiErrors.IsNotFound(err))
}
