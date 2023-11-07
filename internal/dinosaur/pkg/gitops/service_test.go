package gitops

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestRenderCentral(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		params CentralParams
		assert func(t *testing.T, got v1alpha1.Central, err error)
	}{
		{
			name: "no overrides",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{},
			assert: func(t *testing.T, got v1alpha1.Central, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "multiple overrides",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				Centrals: CentralsConfig{
					Overrides: []CentralOverride{
						{
							InstanceIDs: []string{"central-1"},
							Patch:       `metadata: {"labels": {"foo": "bar"}}`,
						}, {
							InstanceIDs: []string{"central-1"},
							Patch:       `metadata: {"annotations": {"foo": "bar"}}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got v1alpha1.Central, err error) {
				require.NoError(t, err)
				assert.Equal(t, "bar", got.Labels["foo"])
				assert.Equal(t, "bar", got.Annotations["foo"])
			},
		},
		{
			name: "multiple overrides, one not matching",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				Centrals: CentralsConfig{
					Overrides: []CentralOverride{
						{
							InstanceIDs: []string{"central-1"},
							Patch:       `metadata: {"labels": {"foo": "bar"}}`,
						}, {
							InstanceIDs: []string{"central-2"},
							Patch:       `metadata: {"labels": {"foo": "baz"}}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got v1alpha1.Central, err error) {
				require.NoError(t, err)
				assert.Equal(t, "bar", got.Labels["foo"])
			},
		},
		{
			name: "with templated patch",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				Centrals: CentralsConfig{
					Overrides: []CentralOverride{
						{
							InstanceIDs: []string{"central-1"},
							Patch:       `metadata: {"labels": {"foo": "{{ .ID }}"}}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got v1alpha1.Central, err error) {
				require.NoError(t, err)
				assert.Equal(t, "central-1", got.Labels["foo"])
			},
		},
		{
			name: "wildcard override",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				Centrals: CentralsConfig{
					Overrides: []CentralOverride{
						{
							InstanceIDs: []string{"*"},
							Patch:       `metadata: {"labels": {"foo": "bar"}}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got v1alpha1.Central, err error) {
				require.NoError(t, err)
				assert.Equal(t, "bar", got.Labels["foo"])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderCentral(tt.params, tt.config)
			tt.assert(t, got, err)
		})
	}
}

// TestDefaultTemplateIsValid tests that the default template is valid and
// can be unmarshaled to a functional v1alpha1.Central object.
func Test_defaultTemplate_isValid(t *testing.T) {

	var wr strings.Builder
	err := defaultTemplate.Execute(&wr, CentralParams{
		ID:               "id",
		Name:             "name",
		Namespace:        "namespace",
		Region:           "region",
		ClusterID:        "cluster-id",
		CloudProvider:    "cloud-provider",
		CloudAccountID:   "cloud-account-id",
		SubscriptionID:   "subscription-id",
		Owner:            "owner",
		OwnerAccountID:   "owner-account-id",
		OwnerUserID:      "owner-user-id",
		Host:             "host",
		OrganizationID:   "organization-id",
		OrganizationName: "organization-name",
		InstanceType:     "instance-type",
		IsInternal:       false,
	})
	require.NoError(t, err)

	var central v1alpha1.Central
	require.NoError(t, yaml.Unmarshal([]byte(wr.String()), &central))
}

type mockProvider struct {
	config Config
}

func (m *mockProvider) Get() (Config, error) {
	return m.config, nil
}

func newMockProvider(config Config) *mockProvider {
	return &mockProvider{config: config}
}
