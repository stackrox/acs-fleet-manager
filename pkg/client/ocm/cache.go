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

type record[V any] struct {
	value V
	ts    time.Time
}

type cacheImpl[K comparable, V any] struct {
	age         time.Duration
	mux         sync.Mutex
	data        map[K]record[V]
	lastCleanUp time.Time
}

// NewCache constructs a key-value cache that stores values up to age period.
func NewCache[K comparable, V any](age time.Duration) *cacheImpl[K, V] {
	return &cacheImpl[K, V]{
		age:  age,
		data: make(map[K]record[V]),
	}
}

// Add a value to the cache. Cleans the cache up from expired data.
// Returns the provided value.
func (c *cacheImpl[K, V]) Add(key K, value V) V {
	c.mux.Lock()
	defer c.mux.Unlock()
	now := time.Now()
	if now.Sub(c.lastCleanUp) > c.age {
		for k, r := range c.data {
			if now.Sub(r.ts) > c.age {
				delete(c.data, k)
			}
		}
		c.lastCleanUp = time.Now()
	}
	c.data[key] = record[V]{value, time.Now()}
	return value
}

// Get the value from the cache. Removes the value from the cache if expired.
// Returns the value and whether it exists in the cache.
func (c *cacheImpl[K, V]) Get(key K) (V, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	rec, ok := c.data[key]
	if ok && time.Since(rec.ts) > c.age {
		delete(c.data, key)
		ok = false
	}
	return rec.value, ok
}
