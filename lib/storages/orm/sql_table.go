package orm

type sqlTable interface {
	CacheKey() string
}
