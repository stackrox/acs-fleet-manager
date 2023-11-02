package gitops

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestService_GetCentral(t *testing.T) {
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
			svc := NewService(newMockProvider(tt.config))
			got, err := svc.GetCentral(tt.params)
			centralGot := v1alpha1.Central{}
			errUnmarshal := yaml.Unmarshal([]byte(got), &centralGot)
			require.NoError(t, errUnmarshal)
			tt.assert(t, centralGot, err)
		})
	}
}

func TestServiceGetCache(t *testing.T) {
	c := Config{
		Centrals: CentralsConfig{
			Overrides: []CentralOverride{
				{
					InstanceIDs: []string{"*"},
					Patch:       `metadata: {"labels": {"foo": "bar"}}`,
				},
			},
		},
	}
	params := CentralParams{
		ID:   "id-123",
		Name: "Central-Name",
	}

	gitOpsService := NewService(newMockProvider(c))
	centralYAML, err := gitOpsService.GetCentral(params)
	require.NoError(t, err)

	central := v1alpha1.Central{}
	err = yaml.Unmarshal([]byte(centralYAML), &central)
	require.NoError(t, err)

	// assert that the rendered Central is present in the cache
	require.Len(t, centralCRYAMLCache, 1)
	key, err := getCacheKey(params, c)
	assert.Equal(t, centralYAML, centralCRYAMLCache[key])
	require.NoError(t, err)

	// assert that the Central is returned correctly from the cache
	centralYAMLFromCache, err := readFromCache(params, c)
	require.NoError(t, err)
	assert.Equal(t, centralYAML, centralYAMLFromCache)

	// assert that the Central is NOT returned with a different config
	c.Centrals = CentralsConfig{}
	result, err := readFromCache(params, c)
	require.NoError(t, err)
	assert.Empty(t, result)
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
