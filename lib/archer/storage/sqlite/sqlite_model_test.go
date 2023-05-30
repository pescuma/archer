package sqlite

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEqualsEmpty(t *testing.T) {
	p1 := &sqlProject{}
	p2 := &sqlProject{}

	assert.True(t, reflect.DeepEqual(p1, p2))

	p1.Name = "a"
	assert.False(t, reflect.DeepEqual(p1, p2))
}

func TestEqualsSomeFields(t *testing.T) {
	now := time.Now()
	p1 := &sqlProject{
		ID:        "a",
		CreatedAt: now,
	}
	p2 := &sqlProject{
		ID:        "a",
		CreatedAt: now,
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

	p1.Name = "b"
	assert.False(t, reflect.DeepEqual(p1, p2))
}

func TestEqualsSize(t *testing.T) {
	p1 := &sqlProject{
		Sizes: map[string]*sqlSize{
			"a": {Bytes: 1},
		},
	}
	p2 := &sqlProject{
		Sizes: map[string]*sqlSize{
			"a": {Bytes: 1},
		},
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

	p1.Sizes["a"].Bytes = 2
	assert.False(t, reflect.DeepEqual(p1, p2))
}

func TestEqualsData(t *testing.T) {
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
