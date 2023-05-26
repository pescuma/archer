package utils

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/constraints"
)

func Take[T any](l []T, i int) []T {
	if i < 0 {
		i = Max(0, len(l)-1+i)
	} else {
		i = Min(i, len(l))
	}
	return l[:i]
}

func First[T any](l []T) T {
	return l[0]
}

func Last[T any](l []T) T {
	return l[len(l)-1]
}

func RemoveLast[T any](l []T) []T {
	return l[:len(l)-1]
}

func Min[T constraints.Ordered](a T, bs ...T) T {
	result := a
	for _, b := range bs {
		if result > b {
			result = b
		}
	}
	return result
}

func Max[T constraints.Ordered](a T, bs ...T) T {
	result := a
	for _, b := range bs {
		if result < b {
			result = b
		}
	}
	return result
}

func IIf[T any](test bool, ifTrue, ifFalse T) T {
	if test {
		return ifTrue
	} else {
		return ifFalse
	}
}

func Coalesce[T comparable](vs ...T) T {
	var def T

	for _, v := range vs {
		if v != def {
			return v
		}
	}

	return def
}

func MapContains[K comparable, V any](m map[K]V, k K) bool {
	_, ok := m[k]
	return ok
}

func MapMapContains[K1, K2 comparable, V any](m1 map[K1]map[K2]V, k1 K1, k2 K2) bool {
	m2, ok := m1[k1]
	if !ok {
		return false
	}

	_, ok = m2[k2]

	return ok
}

func mapGetOrUpdate[K comparable, V any](m map[K]V, k K, update func() V) V {
	v, ok := m[k]

	if !ok {
		v = update()
		m[k] = v
	}

	return v
}

func IsTrue(v string) bool {
	v = strings.ToLower(v)
	return v != "false" && v != "f" && v != "no" && v != "n" && v != ""
}

func in[T comparable](el T, options ...T) bool {
	for _, o := range options {
		if el == o {
			return true
		}
	}

	return false
}

func PathAbs(path string) (string, error) {
	if strings.HasPrefix(filepath.ToSlash(path), "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		path = filepath.Join(home, path[2:])
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return path, nil
}
