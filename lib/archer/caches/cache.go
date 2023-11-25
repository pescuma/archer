package caches

type Cache[K comparable, V any] interface {
	Get(key K, loader func(K) (V, error)) (V, error)
}
