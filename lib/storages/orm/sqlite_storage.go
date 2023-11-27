package orm

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func WithSqlite(file string) gorm.Dialector {
	return sqlite.Open(file + "?_pragma=journal_mode(WAL)")
}

func WithSqliteInMemory() gorm.Dialector {
	return sqlite.Open(":memory:")
}
