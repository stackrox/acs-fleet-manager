package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CreateNonEmptyFile creates non-empty test file.
func CreateNonEmptyFile(t *testing.T) *os.File {
	file, err := os.CreateTemp(t.TempDir(), "test-non-empty-file-")
	assert.NoError(t, err)
	_, err = file.Write([]byte("mock-value"))
	assert.NoError(t, err)
	return file
}
