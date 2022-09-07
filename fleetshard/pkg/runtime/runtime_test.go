package runtime

import (
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRuntime(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	fleetManagerClientMock := &fleetmanager.FleetManagerClientMock{}
	fleetManagerClientMock.CreateCentralFunc = func(request public.CentralRequestPayload) (*public.CentralRequest, error) {
		return &public.CentralRequest{
			Id:      "id",
			Kind:    "",
			MultiAz: true,
			Region:  "us-east1",
			Version: "3.70",
		}, nil
	}

	runtime := Runtime{
		k8sClient: fakeClient,
		client:    fleetManagerClientMock,
		config: &config.Config{
			CreateAuthProvider: false,
		},
	}
	err := runtime.Start()
	require.NoError(t, err)
}
