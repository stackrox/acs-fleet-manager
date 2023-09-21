package ocm

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] interface {
	Add(k K, v V) V
	Get(k K) (V, bool)
}

type cacheImpl[K comparable, V any] struct {
	age time.Duration
	mux sync.Mutex
	m   map[K]V
	ts  map[K]time.Time
}

func NewCache[K comparable, V any](age time.Duration) *cacheImpl[K, V] {
	return &cacheImpl[K, V]{
		age: age,
		m:   make(map[K]V),
		ts:  make(map[K]time.Time),
	}
}

func (c *cacheImpl[K, V]) Add(k K, v V) V {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.m[k] = v
	c.ts[k] = time.Now()
	return v
}

func (c *cacheImpl[K, V]) Get(k K) (V, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	v, ok := c.m[k]
	if ok && time.Since(c.ts[k]) > c.age {
		delete(c.m, k)
		delete(c.ts, k)
		ok = false
	}
	return v, ok
}
