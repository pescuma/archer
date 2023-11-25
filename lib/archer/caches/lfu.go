package caches

import (
	"sync"
)

type Options struct {
	MaxSize      int
	LevelFactor  int
	LevelMinSize int
}

type LFU[K comparable, V any] struct {
	mutex sync.RWMutex
	opts  Options
	m     map[K]*entry[K, V]
	head  level[K, V]
	tail  level[K, V]
}

func NewLFU[K comparable, V any](opts ...Options) Cache[K, V] {
	o := Options{
		MaxSize:      1000,
		LevelFactor:  10,
		LevelMinSize: -1,
	}
	for _, opt := range opts {
		if opt.MaxSize > 0 {
			o.MaxSize = opt.MaxSize
		}
		if opt.LevelFactor > 0 {
			o.LevelFactor = opt.LevelFactor
		}
		if opt.LevelMinSize > 0 {
			o.LevelMinSize = opt.LevelMinSize
		}
	}
	if o.LevelMinSize == -1 {
		o.LevelMinSize = o.MaxSize / 10
	}

	result := &LFU[K, V]{
		opts: o,
		m:    make(map[K]*entry[K, V]),
	}

	lvl := newLevel[K, V](10)
	lvl.insert(&result.head, &result.tail)

	return result
}

func (c *LFU[K, V]) Get(key K, loader func(K) (V, error)) (V, error) {
	c.mutex.Lock()

	entry, ok := c.m[key]
	if !ok {
		entry = c.newEntry(key, NewLazy[V](func() (V, error) { return loader(key) }))
		c.m[key] = entry
	} else {
		c.incUsage(entry)
	}

	c.mutex.Unlock()

	return entry.val.Get()
}

func (c *LFU[K, V]) newEntry(key K, val *Lazy[V]) *entry[K, V] {
	if len(c.m) >= c.opts.MaxSize {
		lvl := c.findLevelToRemoveEntry()

		last := lvl.tail.prev
		delete(lvl.entries, last)
		delete(c.m, last.key)
		last.remove()

		if len(lvl.entries) == 0 && lvl.max > c.opts.LevelFactor {
			lvl.remove()
		}
	}

	if c.head.next.max > c.opts.LevelFactor {
		lvl := newLevel[K, V](c.opts.LevelFactor)
		lvl.insert(&c.head, c.head.next)
	}

	lvl := c.head.next

	result := &entry[K, V]{
		usages: 1,
		key:    key,
		val:    val,
	}
	result.insert(lvl, &lvl.head, lvl.head.next)
	lvl.entries[result] = true

	return result
}

func (c *LFU[K, V]) findLevelToRemoveEntry() *level[K, V] {
	// The intention is to give new entries some time to be used before they are removed
	for lvl := c.head.next; lvl != &c.tail; lvl = lvl.next {
		if len(lvl.entries) > c.opts.LevelMinSize {
			return lvl
		}
	}

	return c.head.next
}

func (c *LFU[K, V]) incUsage(e *entry[K, V]) {
	lvl := e.lvl
	e.usages++

	if e.usages < lvl.max {
		e.remove()
		e.insert(lvl, &lvl.head, lvl.head.next)

	} else {
		nextMax := lvl.max * c.opts.LevelFactor

		next := lvl.next
		if next.max != nextMax {
			next = newLevel[K, V](nextMax)
			next.insert(lvl, lvl.next)
		}

		delete(lvl.entries, e)
		e.remove()

		if len(lvl.entries) == 0 {
			lvl.remove()
		}

		e.insert(next, &next.head, next.head.next)
		next.entries[e] = true
	}
}

func (c *LFU[K, V]) Len() int {
	return len(c.m)
}
