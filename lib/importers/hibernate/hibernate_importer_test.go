package hibernate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanTypeName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Abc", cleanTypeName("Abc"))
	assert.Equal(t, "Abc", cleanTypeName("Abc?"))
	assert.Equal(t, "Abc", cleanTypeName("ListRepositories<Abc>"))
	assert.Equal(t, "Abc", cleanTypeName("MutableList<Abc>"))
	assert.Equal(t, "Abc", cleanTypeName("Set<Abc>"))
	assert.Equal(t, "Abc", cleanTypeName("MutableSet<Abc>"))
	assert.Equal(t, "Abc", cleanTypeName("ListRepositories<Abc>?"))
	assert.Equal(t, "Abc", cleanTypeName("MutableList<Abc>?"))
	assert.Equal(t, "Abc", cleanTypeName("Set<Abc>?"))
	assert.Equal(t, "Abc", cleanTypeName("MutableSet<Abc>?"))
	assert.Equal(t, "Abc", cleanTypeName("ListRepositories<Abc?>"))
	assert.Equal(t, "Abc", cleanTypeName("MutableList<Abc?>"))
	assert.Equal(t, "Abc", cleanTypeName("Set<Abc?>"))
	assert.Equal(t, "Abc", cleanTypeName("MutableSet<Abc?>"))
}
