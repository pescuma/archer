package kotlin

import (
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/archer/languages/kotlin_parser"
	"github.com/pescuma/archer/lib/archer/stucture"
)

func computeStructure(contents string) *stucture.FileStructure {
	input := antlr.NewInputStream(contents)
	lexer := kotlin_parser.NewKotlinLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := kotlin_parser.NewKotlinParser(stream)

	return ImportStructure("a.kt", parser.KotlinFile())
}

func TestEmptyFile(t *testing.T) {
	t.Parallel()

	structure := computeStructure("")

	assert.Equal(t, 0, len(structure.AllClasses))
	assert.Equal(t, 0, len(structure.AllFunctions))
}
