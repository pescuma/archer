package caches

import (
	"sync"
)

type Cache[K comparable, V any] struct {
	mutex sync.RWMutex
	m     map[K]*Lazy[V]
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	result := &Cache[K, V]{
		m: make(map[K]*Lazy[V], 10000),
	}

	return result
}

func (c *Cache[K, V]) Get(key K, loader func(K) (V, error)) (V, error) {
	c.mutex.RLock()
	val, ok := c.m[key]
	c.mutex.RUnlock()

	if ok {
		return val.Get()
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	val, ok = c.m[key]
	if !ok {
		val = NewLazy[V](func() (V, error) { return loader(key) })
		c.m[key] = val
	}

	return val.Get()
}

func (c *Cache[K, V]) Len() int {
	return len(c.m)
}
