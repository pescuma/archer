package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/importers/metrics"
	"github.com/pescuma/archer/lib/workspace"
)

func TestCognitiveNoCode(t *testing.T) {
	t.Parallel()

	mif := 200

	ws, err := workspace.NewWorkspace(":memory:")
	assert.Nil(t, err)

	err = ws.ImportMetrics(nil, &metrics.Options{
		Incremental:      true,
		MaxImportedFiles: &mif,
	})
	assert.Nil(t, err)
}
