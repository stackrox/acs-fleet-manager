package ocm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Run("simple add and get", func(t *testing.T) {
		c := NewCache[string, string](time.Minute)
		a := c.Add("a", "ay")
		assert.Equal(t, "ay", a)

		a, ok := c.Get("a")
		assert.Equal(t, "ay", a)
		assert.True(t, ok)

		b, ok := c.Get("b")
		assert.Equal(t, "", b)
		assert.False(t, ok)
	})

	t.Run("simple override", func(t *testing.T) {
		c := NewCache[string, string](time.Minute)
		c.Add("a", "ay")
		c.Add("a", "oy")
		a, ok := c.Get("a")
		assert.Equal(t, "oy", a)
		assert.True(t, ok)
	})

	t.Run("delete expired", func(t *testing.T) {
		c := NewCache[string, string](0)
		c.Add("a", "ay")
		a, ok := c.Get("a")
		assert.Equal(t, "ay", a)
		assert.False(t, ok)

		a, ok = c.Get("a")
		assert.Equal(t, "", a)
		assert.False(t, ok)
	})

	t.Run("cleanup", func(t *testing.T) {
		c := NewCache[string, string](0)
		c.Add("a", "ay")
		c.Add("b", "bee")
		a, ok := c.Get("a")
		assert.Equal(t, "", a)
		assert.False(t, ok)

		b, ok := c.Get("b")
		assert.Equal(t, "bee", b)
		assert.False(t, ok)
	})

}
