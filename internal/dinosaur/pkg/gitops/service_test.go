package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTenantResourceValues(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		params CentralParams
		assert func(t *testing.T, got map[string]interface{}, err error)
	}{
		{
			name: "no overrides",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				assert.NoError(t, err)
				assert.Empty(t, got)
			},
		}, {
			name: "default without overrides",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				TenantResources: TenantResourceConfig{
					Default: `{"foo": "bar"}`,
				},
			},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]interface{}{"foo": "bar"}, got)
			},
		}, {
			name: "default with override",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				TenantResources: TenantResourceConfig{
					Default: `{"foo": "bar"}`,
					Overrides: []TenantResourceOverride{
						{
							InstanceIDs: []string{"central-1"},
							Values:      `{"foo": "baz"}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]interface{}{"foo": "baz"}, got)
			},
		},
		{
			name: "default with multiple overrides",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				TenantResources: TenantResourceConfig{
					Default: `{"foo": "bar"}`,
					Overrides: []TenantResourceOverride{
						{
							InstanceIDs: []string{"central-1"},
							Values:      `{"foo": "baz"}`,
						},
						{
							InstanceIDs: []string{"central-1"},
							Values:      `{"foo": "qux"}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]interface{}{"foo": "qux"}, got)
			},
		}, {
			name: "complex valued patch",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				TenantResources: TenantResourceConfig{
					Default: `{"buzz":"snap", "foo": "snafu"}`,
					Overrides: []TenantResourceOverride{
						{
							InstanceIDs: []string{"central-1"},
							Values:      `{"foo":{ "bar": "qux" }}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]interface{}{"buzz": "snap", "foo": map[string]interface{}{"bar": "qux"}}, got)
			},
		}, {
			name: "default with templating",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				TenantResources: TenantResourceConfig{
					Default: `{"foo": "{{ .ID }}"}`,
				},
			},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]interface{}{"foo": "central-1"}, got)
			},
		}, {
			name: "overrides with templating",
			params: CentralParams{
				ID: "central-1",
			},
			config: Config{
				TenantResources: TenantResourceConfig{
					Default: `{"foo": "bar"}`,
					Overrides: []TenantResourceOverride{
						{
							InstanceIDs: []string{"central-1"},
							Values:      `{"foo": "{{ .ID }}"}`,
						},
					},
				},
			},
			assert: func(t *testing.T, got map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]interface{}{"foo": "central-1"}, got)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderTenantResourceValues(tt.params, tt.config)
			tt.assert(t, got, err)
		})
	}
}
