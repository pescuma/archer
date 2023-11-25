package utils

import (
	"sync"
)

type Lazy[T any] struct {
	mutex  sync.RWMutex
	loaded bool
	loader func() (T, error)
	val    T
	err    error
}

func NewLazy[T any](loader func() (T, error)) *Lazy[T] {
	return &Lazy[T]{
		loader: loader,
	}
}

func (l *Lazy[T]) Get() (T, error) {
	l.mutex.RLock()
	loaded := l.loaded
	l.mutex.RUnlock()

	if loaded {
		return l.val, l.err
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.loaded {
		return l.val, l.err
	}

	l.val, l.err = l.loader()
	l.loaded = true

	return l.val, l.err
}

type Cache[K comparable, V any] struct {
	mutex sync.RWMutex
	opts  CacheOptions
	m     map[K]*Lazy[V]
	head  node[V]
	tail  node[V]
}

type CacheOptions struct {
	MaxSize int
}

type node[V any] struct {
	prev *node[V]
	next *node[V]
	uses int
	val  *Lazy[V]
}

func NewCache[K comparable, V any](opts ...CacheOptions) *Cache[K, V] {
	o := CacheOptions{}
	for _, opt := range opts {
		if opt.MaxSize > 0 {
			o.MaxSize = opt.MaxSize
		}
	}

	result := &Cache[K, V]{
		opts: o,
		m:    make(map[K]*Lazy[V], 10000),
	}
	result.head.next = &result.tail
	result.tail.prev = &result.head

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
