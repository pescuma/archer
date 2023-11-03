package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/importers/metrics"
	"github.com/pescuma/archer/lib/archer/storage/sqlite"
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
