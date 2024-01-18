package workers

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestClusterManager_processReadyClusters_emptyConfig(t *testing.T) {
	gitopsConfig := gitops.Config{}
	provider := mockProvider{config: gitopsConfig}
	clusterService := &services.ClusterServiceMock{
		ListByStatusFunc: func(state api.ClusterStatus) ([]api.Cluster, *errors.ServiceError) {
			if state == api.ClusterReady {
				return []api.Cluster{
					{
						ClusterID: "1234567890abcdef1234567890abcdef", // pragma: allowlist secret
					},
				}, nil
			}
			return []api.Cluster{}, nil
		},
	}
	c := &ClusterManager{
		ClusterManagerOptions: ClusterManagerOptions{
			ClusterService:         clusterService,
			GitOpsConfigProvider:   provider,
			DataplaneClusterConfig: &config.DataplaneClusterConfig{},
		},
	}
	errs := c.processReadyClusters()
	assert.Empty(t, errs)
}

type mockProvider struct {
	config gitops.Config
}

func (m mockProvider) Get() (gitops.Config, error) {
	return m.config, nil
}
