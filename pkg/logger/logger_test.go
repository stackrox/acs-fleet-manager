package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfoChangedInt32(t *testing.T) {
	times := 0

	increment := func() { times = times + 1 }

	counter := int32(42)
	doIfChangedInt32(&counter, increment) // log

	counter = int32(43)
	doIfChangedInt32(&counter, increment) // log

	counter = int32(43)
	doIfChangedInt32(&counter, increment) // skip

	assert.Equal(t, 2, times)
}
