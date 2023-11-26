package caches

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
