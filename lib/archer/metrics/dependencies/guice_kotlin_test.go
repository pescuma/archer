package dependencies

import (
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/stretchr/testify/assert"

	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
)

func computeGuiceDeps(contents string) int {
	input := antlr.NewInputStream(contents)
	lexer := kotlin_parser.NewKotlinLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)

	p := kotlin_parser.NewKotlinParser(stream)

	file := p.KotlinFile()

	return ComputeKotlinGuiceDependencies("a.kt", nil, file)
}

func TestEmptyFile(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("")

	assert.Equal(t, 0, deps)
}

func TestConstructorWithoutGuice(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("class A(private val a: B) {}")

	assert.Equal(t, 0, deps)
}

func TestConstructorWithGuice(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("class A @Inject constructor(private val a: B) {}")

	assert.Equal(t, 1, deps)
}

func TestSecondaryConstructorWithoutGuice(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("class A{ constructor(a: B) {} }")

	assert.Equal(t, 0, deps)
}

func TestSecondaryConstructorWithGuice(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("class A{ @Inject constructor(a: B) {} }")

	assert.Equal(t, 1, deps)
}

func TestPropertyWithoutGuice(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("class A{ private var a: B }")

	assert.Equal(t, 0, deps)
}

func TestPropertyWithGuice(t *testing.T) {
	t.Parallel()

	deps := computeGuiceDeps("class A{ @Inject private var a: B }")

	assert.Equal(t, 1, deps)
}
