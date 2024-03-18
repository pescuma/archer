package orm

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEqualsEmpty(t *testing.T) {
	t.Parallel()

	p1 := &sqlProject{}
	p2 := &sqlProject{}

	assert.True(t, reflect.DeepEqual(p1, p2))

	p1.Name = "a"
	assert.False(t, reflect.DeepEqual(p1, p2))
}

func TestEqualsSomeFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	p1 := &sqlProject{
		ID:        1,
		CreatedAt: now,
	}
	p2 := &sqlProject{
		ID:        1,
		CreatedAt: now,
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

	p1.Name = "b"
	assert.False(t, reflect.DeepEqual(p1, p2))
}

func TestEqualsSize(t *testing.T) {
	t.Parallel()

	v1 := 1
	p1 := &sqlProject{
		Sizes: map[string]*sqlSize{
			"a": {Bytes: &v1},
		},
	}

	v2 := 1
	p2 := &sqlProject{
		Sizes: map[string]*sqlSize{
			"a": {Bytes: &v2},
		},
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

	v3 := 2
	p1.Sizes["a"].Bytes = &v3
	assert.False(t, reflect.DeepEqual(p1, p2))
}

func TestEqualsData(t *testing.T) {
	t.Parallel()

	p1 := &sqlProject{
		Data: map[string]string{
			"a": "b",
		},
	}
	p2 := &sqlProject{
		Data: map[string]string{
			"a": "b",
		},
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

	p1.Data["a"] = "c"
	assert.False(t, reflect.DeepEqual(p1, p2))
}
