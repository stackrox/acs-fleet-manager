package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsStructWithoutPointers(t *testing.T) {
	type structWithPointers struct {
		_ *int
		_ int
		_ *time.Time
	}
	type structWithoutPointers struct {
		_ int
		_ string
		_ sql.NullTime
	}
	assert.False(t, IsStructWithoutPointers(structWithPointers{}))
	assert.True(t, IsStructWithoutPointers(structWithoutPointers{}))
}
