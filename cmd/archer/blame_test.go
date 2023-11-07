package main

import (
	"testing"

	"github.com/pescuma/archer/lib/archer/importers/git"
	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/archer"
	"github.com/pescuma/archer/lib/archer/storage/sqlite"
)

func TestBlame(t *testing.T) {
	t.Parallel()

	mif := 10
	g := git.NewBlameImporter([]string{"C:\\devel\\kopia"}, git.BlameOptions{
		Incremental:      true,
		MaxImportedFiles: &mif,
	})

	ws, err := archer.NewWorkspace(sqlite.NewSqliteStorage, "")
	assert.Nil(t, err)

	err = ws.Import(g)
	assert.Nil(t, err)
}
