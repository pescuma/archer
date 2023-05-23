package metrics

import (
	"testing"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/stretchr/testify/assert"

	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
)

func compute(contents string) (int, error) {
	input := antlr.NewInputStream(contents)
	lexer := kotlin_parser.NewKotlinLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)

	p := kotlin_parser.NewKotlinParser(stream)

	file := p.KotlinFile()

	return ComputeKotlinGuiceDependencies("a.kt", file)
}

func TestEmptyFile(t *testing.T) {
	t.Parallel()

	deps, err := compute("")

	assert.Nil(t, err)
	assert.Equal(t, 0, deps)
}

func TestConstructorWithoutGuice(t *testing.T) {
	t.Parallel()

	deps, err := compute("class A(private val a: B) {}")

	assert.Nil(t, err)
	assert.Equal(t, 0, deps)
}

func TestConstructorWithGuice(t *testing.T) {
	t.Parallel()

	deps, err := compute("class A @Inject constructor(private val a: B) {}")

	assert.Nil(t, err)
	assert.Equal(t, 1, deps)
}

func TestSecondaryConstructorWithoutGuice(t *testing.T) {
	t.Parallel()

	deps, err := compute("class A{ constructor(a: B) {} }")

	assert.Nil(t, err)
	assert.Equal(t, 0, deps)
}

func TestSecondaryConstructorWithGuice(t *testing.T) {
	t.Parallel()

	deps, err := compute("class A{ @Inject constructor(a: B) {} }")

	assert.Nil(t, err)
	assert.Equal(t, 1, deps)
}

func TestPropertyWithoutGuice(t *testing.T) {
	t.Parallel()

	deps, err := compute("class A{ private var a: B }")

	assert.Nil(t, err)
	assert.Equal(t, 0, deps)
}

func TestPropertyWithGuice(t *testing.T) {
	t.Parallel()

	deps, err := compute("class A{ @Inject private var a: B }")

	assert.Nil(t, err)
	assert.Equal(t, 1, deps)
}
