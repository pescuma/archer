package complexity

import (
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/archer/languages/kotlin_parser"
)

func computeCognitive(contents string) int {
	input := antlr.NewInputStream(contents)
	lexer := kotlin_parser.NewKotlinLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)

	p := kotlin_parser.NewKotlinParser(stream)

	file := p.KotlinFile()

	return ComputeKotlinComplexity("a.kt", file).CognitiveComplexity
}

func TestCognitiveNoCode(t *testing.T) {
	t.Parallel()

	deps := computeCognitive("class A { fun b() {} }")

	assert.Equal(t, 0, deps)
}

func TestCognitiveElseIf(t *testing.T) {
	t.Parallel()

	deps := computeCognitive(`
class A { 
    fun b() { 
        if (i == 1) { 
        } else if (i == 1) {
        } 
    } 
}
`)

	assert.Equal(t, 2, deps)
}

func TestCognitiveElseBlockIf(t *testing.T) {
	t.Parallel()

	deps := computeCognitive(`
class A { 
    fun b() { 
        if (i == 1) { 
        } else {
			if (i == 1) {
			}
        } 
    } 
}
`)

	assert.Equal(t, 3, deps)
}

func TestCognitiveBreak(t *testing.T) {
	t.Parallel()

	deps := computeCognitive(`
class A { 
    fun b() { 
		loop@ for (i in 1..100) {
			break@loop
		}
    } 
}
`)

	assert.Equal(t, 2, deps)
}

func TestCognitiveRecursiveCall(t *testing.T) {
	t.Parallel()

	deps := computeCognitive(`class A { fun b() { b() } }`)

	assert.Equal(t, 1, deps)
}
