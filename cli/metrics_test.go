package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/importers/metrics"
	"github.com/pescuma/archer/lib/storages/sqlite"
	"github.com/pescuma/archer/lib/workspace"
)

func TestCognitiveNoCode(t *testing.T) {
	t.Parallel()

	mif := 200

	g := metrics.NewImporter(nil, metrics.Options{
		Incremental:      true,
		MaxImportedFiles: &mif,
	})

	ws, err := workspace.NewWorkspace(sqlite.NewSqliteMemoryStorage, "")
	assert.Nil(t, err)

	err = ws.Import(g)
	assert.Nil(t, err)
}
