package gitops

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileReader_Get_FailsIfFileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := tmpDir + "/config.yaml"
	provider := NewFileReader(tmpFilePath)
	_, err := provider.Read()
	assert.Error(t, err)
}

func TestFileReader_Get_FailsIfFileIsInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := tmpDir + "/config.yaml"
	file, err := os.Create(tmpFilePath)
	require.NoError(t, err)
	_, err = file.WriteString("invalid yaml")
	require.NoError(t, err)
	provider := NewFileReader(tmpFilePath)
	_, err = provider.Read()
	assert.Error(t, err)
}

func TestFileReader_Get(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFilePath := tmpDir + "/config.yaml"
	file, err := os.Create(tmpFilePath)
	require.NoError(t, err)
	_, err = file.WriteString(`
centrals:
  overrides: []
`)
	require.NoError(t, err)
	provider := NewFileReader(tmpFilePath)
	_, err = provider.Read()
	require.NoError(t, err)
}

func TestStaticReader_Read(t *testing.T) {
	provider := NewStaticReader(Config{})
	_, err := provider.Read()
	require.NoError(t, err)
}
