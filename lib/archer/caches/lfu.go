package caches

import (
	"sync"
)

type Options struct {
	MaxSize int
}

type LFU[K comparable, V any] struct {
	mutex sync.RWMutex
	opts  Options
	m     map[K]*entry[V]
	head  level[V]
	tail  level[V]
}

// levels store entries with similar numbers of usages
// they grow exponentially in size
type level[V any] struct {
	prev *level[V]
	next *level[V]

	max     int
	entries map[*entry[V]]bool
	head    entry[V]
	tail    entry[V]
}

func newLevel[V any](max int) *level[V] {
	return &level[V]{
		max: max,
	}
}
func (l *level[V]) insert(a, b *level[V]) {
	a.next = l
	l.prev = a

	l.next = b
	b.prev = l
}
func (l *level[V]) remove() {
	l.prev.next = l.next
	l.next.prev = l.prev
}

type entry[V any] struct {
	prev *entry[V]
	next *entry[V]

	lvl    *level[V]
	usages int
	val    *Lazy[V]
}

func (e *entry[V]) insert(a, b *entry[V]) {
	a.next = e
	e.prev = a

	e.next = b
	b.prev = e
}
func (e *entry[V]) remove() {
	e.prev.next = e.next
	e.next.prev = e.prev
}

func NewLFU[K comparable, V any](opts ...Options) *LFU[K, V] {
	o := Options{
		MaxSize: 1000,
	}
	for _, opt := range opts {
		if opt.MaxSize > 0 {
			o.MaxSize = opt.MaxSize
		}
	}

	result := &LFU[K, V]{
		opts: o,
		m:    make(map[K]*entry[V]),
	}

	lvl := newLevel[V](10)
	lvl.insert(&result.head, &result.tail)

	return result
}

func (c *LFU[K, V]) Get(key K, loader func(K) (V, error)) (V, error) {
	c.mutex.Lock()

	entry, ok := c.m[key]
	if !ok {
		entry = c.newEntry(NewLazy[V](func() (V, error) { return loader(key) }))
		c.m[key] = entry
	} else {
		c.incUsage(entry)
	}

	c.mutex.Unlock()

	return entry.val.Get()
}

func (c *LFU[K, V]) newEntry(val *Lazy[V]) *entry[V] {
	if len(c.m) >= c.opts.MaxSize {
		lvl := c.head.next

		last := lvl.tail.prev
		delete(lvl.entries, last)
		last.remove()

		if len(lvl.entries) == 0 && lvl.max > 10 {
			lvl.remove()
		}
	}

	if c.head.next.max > 10 {
		lvl := newLevel[V](10)
		lvl.insert(&c.head, c.head.next)
	}

	lvl := c.head.next

	result := &entry[V]{
		lvl:    lvl,
		usages: 1,
		val:    val,
	}
	lvl.entries[result] = true
	result.insert(&lvl.head, lvl.head.next)

	return result
}

func (c *LFU[K, V]) incUsage(e *entry[V]) {
	lvl := e.lvl
	e.usages++

	if e.usages < lvl.max {
		e.remove()
		e.insert(&lvl.head, lvl.head.next)

	} else {
		nextMax := lvl.max * 10

		next := lvl.next
		if next.max != nextMax {
			next = newLevel[V](nextMax)
			next.insert(lvl, lvl.next)
		}

		delete(lvl.entries, e)
		e.remove()

		if len(lvl.entries) == 0 {
			lvl.remove()
		}

		e.lvl = next
		next.entries[e] = true
		e.insert(&next.head, next.head.next)
	}
}

func (c *LFU[K, V]) Len() int {
	return len(c.m)
}
