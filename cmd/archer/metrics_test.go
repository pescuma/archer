package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/importers/metrics"
	"github.com/Faire/archer/lib/archer/storage/sqlite"
)

func TestCognitiveNoCode(t *testing.T) {
	t.Parallel()

	mif := 200

	g := metrics.NewImporter(nil, metrics.Options{
		Incremental:      true,
		MaxImportedFiles: &mif,
	})

	ws, err := archer.NewWorkspace(sqlite.NewSqliteStorage, "")
	assert.Nil(t, err)

	err = ws.Import(g)
	assert.Nil(t, err)
}
