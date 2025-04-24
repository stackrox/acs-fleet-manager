package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func has[T comparable](array []T, value T) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}
	return false
}

func TestActiveStatuses(t *testing.T) {
	as := ActiveStatuses
	assert.Len(t, as, 4)

	for _, v := range []CentralStatus{"accepted", "preparing", "provisioning", "ready"} {
		assert.True(t, has(as, v))
	}
}
