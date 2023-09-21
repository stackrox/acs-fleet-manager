package ocm

import (
	"sync"
	"time"
)

// Cache provides methods to store and get values from a cache.
type Cache[K comparable, V any] interface {
	Add(key K, value V) V
	Get(key K) (V, bool)
}

type cacheImpl[K comparable, V any] struct {
	age  time.Duration
	mux  sync.Mutex
	data map[K]V
	ts   map[K]time.Time
}

// NewCache constructs a key-value cache that stores values up to age period.
func NewCache[K comparable, V any](age time.Duration) *cacheImpl[K, V] {
	return &cacheImpl[K, V]{
		age:  age,
		data: make(map[K]V),
		ts:   make(map[K]time.Time),
	}
}

// Add a value to the cache. Cleans the cache up from expired data.
// Returns the provided value.
func (c *cacheImpl[K, V]) Add(key K, value V) V {
	c.mux.Lock()
	defer c.mux.Unlock()
	for k, ts := range c.ts {
		if time.Since(ts) > c.age {
			delete(c.data, k)
			delete(c.ts, k)
		}
	}
	c.data[key] = value
	c.ts[key] = time.Now()
	return value
}

// Get the value from the cache. Removes the value from the cache if expired.
// Returns the value and whether it exists in the cache.
func (c *cacheImpl[K, V]) Get(key K) (V, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data[key]
	if ok && time.Since(c.ts[key]) > c.age {
		delete(c.data, key)
		delete(c.ts, key)
		ok = false
	}
	return value, ok
}
