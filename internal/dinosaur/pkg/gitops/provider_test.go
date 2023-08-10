package gitops

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileConfigProvider_Get_FailsIfFileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := tmpDir + "/config.yaml"
	provider := NewFileConfigProvider(tmpFilePath)
	_, err := provider.Get()
	assert.Error(t, err)
}

func TestFileConfigProvider_Get_FailsIfFileIsInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := tmpDir + "/config.yaml"
	file, err := os.Create(tmpFilePath)
	require.NoError(t, err)
	_, err = file.WriteString("invalid yaml")
	require.NoError(t, err)
	provider := NewFileConfigProvider(tmpFilePath)
	_, err = provider.Get()
	assert.Error(t, err)
}

func TestFileConfigProvider_Get(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := tmpDir + "/config.yaml"
	file, err := os.Create(tmpFilePath)
	require.NoError(t, err)
	_, err = file.WriteString(`
centrals:
  overrides:
  - instanceIds: [a, b]
    patch: |
      {}
`)
	require.NoError(t, err)
	provider := NewFileConfigProvider(tmpFilePath)
	config, err := provider.Get()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, config.Centrals.Overrides[0].InstanceIDs)
}

func TestStaticConfigProvider_Get(t *testing.T) {
	provider := NewStaticConfigProvider(Config{
		Centrals: CentralsConfig{
			Overrides: []CentralOverride{
				{
					InstanceIDs: []string{"a", "b"},
					Patch:       "{}",
				},
			},
		},
	})
	config, err := provider.Get()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, config.Centrals.Overrides[0].InstanceIDs)
}

func TestEmptyConfigProvider_Get(t *testing.T) {
	provider := NewEmptyConfigProvider()
	config, err := provider.Get()
	require.NoError(t, err)
	assert.Equal(t, Config{}, config)
}

func TestFallbackToLastWorkingConfigProvider_Get_ShouldFailIfNoLastConfig(t *testing.T) {
	primary := NewMockProvider().WillFail()
	provider := NewFallbackToLastWorkingConfigProvider(primary)
	_, err := provider.Get()
	assert.Error(t, err)
}

func TestFallbackToLastWorkingConfigProvider_Get_ShouldReturnLast(t *testing.T) {
	primary := NewMockProvider()
	provider := NewFallbackToLastWorkingConfigProvider(primary)
	config, err := provider.Get()
	require.NoError(t, err)
	assert.Equal(t, Config{}, config)

	primary.WillFail()
	config, err = provider.Get()
	require.NoError(t, err)
	assert.Equal(t, Config{}, config)
}

func TestProviderWithMetrics_Get(t *testing.T) {
	primary := NewMockProvider()
	provider := NewProviderWithMetrics(primary)
	_, err := provider.Get()
	require.NoError(t, err)
	count := testutil.CollectAndCount(errorCounter)
	assert.Equal(t, 0, count)
}

func TestProviderWithMetrics_Get_ShouldIncrementErrorCounter(t *testing.T) {
	primary := NewMockProvider().WillFail()
	provider := NewProviderWithMetrics(primary)
	_, err := provider.Get()
	require.Error(t, err)
	count := testutil.CollectAndCount(errorCounter)
	assert.Equal(t, 1, count)
}

type mockProvider struct {
	config Config
	err    error
}

func (p *mockProvider) Get() (Config, error) {
	return p.config, p.err
}

func (p *mockProvider) WillFail() *mockProvider {
	p.err = assert.AnError
	return p
}

func (p *mockProvider) WillSucceed() *mockProvider {
	p.err = nil
	return p
}

func NewMockProvider() *mockProvider {
	return &mockProvider{
		config: Config{},
		err:    nil,
	}
}
