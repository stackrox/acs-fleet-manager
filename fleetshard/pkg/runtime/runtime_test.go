package runtime

import (
	"context"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stretchr/testify/assert"
)

const clusterID = "1234567890abcdef1234567890abcdef" // pragma: allowlist secret

func TestLoadClusterConfig(t *testing.T) {
	clientMock := fleetmanager.NewClientMock()
	clientMock.PrivateAPIMock.GetDataPlaneClusterAgentConfigFunc = func(_ context.Context, _ string) (private.DataplaneClusterAgentConfig, *http.Response, error) {
		return private.DataplaneClusterAgentConfig{}, nil, errors.New("Test error")
	}

	runtime := &Runtime{
		config:            &config.Config{},
		k8sClient:         testutils.NewFakeClientBuilder(t).Build(),
		client:            clientMock.Client(),
		clusterID:         clusterID,
		dbProvisionClient: &cloudprovider.DBClientMock{},
		reconcilers:       make(reconcilerRegistry),
	}
	err := runtime.Start()
	assert.EqualError(t, err, "failed to load cluster configuration: Test error")
}
