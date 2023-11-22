package utils

import "sync"

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
	m sync.Map
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{}
}

func (l *Cache[K, V]) Get(key K, loader func(key K) (V, error)) (V, error) {
	val, _ := l.m.LoadOrStore(key, NewLazy[V](func() (V, error) {
		return loader(key)
	}))
	return val.(*Lazy[V]).Get()
}
