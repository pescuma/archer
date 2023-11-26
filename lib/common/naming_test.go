package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pescuma/archer/lib/common"
	"github.com/pescuma/archer/lib/model"
)

func TestCreateTableNameParts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"a"}, createTableNameParts("a"))
	assert.Equal(t, []string{"a", "a_b"}, createTableNameParts("a_b"))
	assert.Equal(t, []string{"a", "a_b", "a_b_c"}, createTableNameParts("a_b_c"))
}

func createTableNameParts(name string) []string {
	projs := model.NewProjects()

	proj := projs.GetOrCreate("r", name)

	common.CreateTableNameParts(projs.ListProjects(model.FilterAll))

	return proj.NameParts
}
