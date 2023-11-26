package caches

// levels store entries with similar numbers of usages
// they grow exponentially in size
type level[K comparable, V any] struct {
	prev *level[K, V]
	next *level[K, V]

	max     int
	entries map[*entry[K, V]]bool
	head    entry[K, V]
	tail    entry[K, V]
}

func newLevel[K comparable, V any](max int) *level[K, V] {
	result := &level[K, V]{
		max:     max,
		entries: make(map[*entry[K, V]]bool),
	}
	result.head.next = &result.tail
	result.tail.prev = &result.head
	return result
}
func (l *level[K, V]) insert(a, b *level[K, V]) {
	a.next = l
	l.prev = a

	l.next = b
	b.prev = l
}
func (l *level[K, V]) remove() {
	l.prev.next = l.next
	l.next.prev = l.prev
}

type entry[K comparable, V any] struct {
	prev *entry[K, V]
	next *entry[K, V]

	lvl    *level[K, V]
	usages int
	key    K
	val    *Lazy[V]
}

func (e *entry[K, V]) insert(lvl *level[K, V], a, b *entry[K, V]) {
	e.lvl = lvl

	a.next = e
	e.prev = a

	e.next = b
	b.prev = e
}
func (e *entry[K, V]) remove() {
	e.lvl = nil
	e.prev.next = e.next
	e.next.prev = e.prev
}
