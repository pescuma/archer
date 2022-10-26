package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Faire/archer/lib/archer"
	"github.com/Faire/archer/lib/archer/common"
)

func TestCreateTableNameParts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"a"}, createTableNameParts("a"))
	assert.Equal(t, []string{"a", "a_b"}, createTableNameParts("a_b"))
	assert.Equal(t, []string{"a", "a_b", "a_b_c"}, createTableNameParts("a_b_c"))
}

func createTableNameParts(name string) []string {
	projs := archer.NewProjects()

	proj := projs.Get("r", name)

	common.CreateTableNameParts(projs.ListProjects(archer.FilterAll))

	return proj.NameParts
}
