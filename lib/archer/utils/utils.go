package utils

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/aquilax/truncate"
	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-gitignore"
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

func In[T comparable](v T, cs ...T) bool {
	for _, c := range cs {
		if v == c {
			return true
		}
	}

	return false
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

func PathAbs(paths ...string) (string, error) {
	path := filepath.Join(paths...)

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

	if filepath.Separator == '\\' {
		pos := strings.Index(path, ":\\")
		if pos != -1 {
			path = strings.ToUpper(path[:pos]) + path[pos:]
		}
	}

	return path, nil
}

func IsTextReader(reader io.ReadCloser, err error) (bool, error) {
	return IsTextReaderOpts(reader, err, 10)
}

func IsTextReaderOpts(reader io.ReadCloser, err error, sampleLines int) (bool, error) {
	if err != nil {
		return false, err
	}

	defer reader.Close()

	fileScanner := bufio.NewScanner(reader)

	for i := 0; i < sampleLines; i++ {
		if !fileScanner.Scan() {
			return true, nil
		}

		if !utf8.ValidString(fileScanner.Text()) {
			return false, nil
		}
	}

	return true, err
}

func IsTextFile(path string) (bool, error) {
	return IsTextReader(os.Open(path))
}

func FileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil

	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil

	} else {
		return false, err
	}
}

func ListFilesRecursive(rootDir string, matcher func(name string) bool) ([]string, error) {
	result := make([]string, 0, 100)

	err := filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil

		case entry.IsDir() && strings.HasPrefix(entry.Name(), "."):
			return filepath.SkipDir

		case !entry.IsDir() && matcher(entry.Name()):
			path, err = PathAbs(path)
			if err != nil {
				return err
			}

			result = append(result, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func FindGitIgnore(path string) (func(string) bool, error) {
	for {
		file := filepath.Join(path, ".gitignore")

		exists, err := FileExists(file)
		if err != nil {
			return nil, err
		}

		if !exists {
			parent := filepath.Dir(path)
			if parent == "." || parent == string(filepath.Separator) || parent == path {
				return nil, nil
			}

			path = parent
			continue
		}

		gi, err := ignore.CompileIgnoreFile(file)
		if err != nil {
			return nil, err
		}

		return func(inner string) bool {
			rel, err := filepath.Rel(path, inner)
			if err != nil {
				return false
			}

			if strings.HasSuffix(inner, string(filepath.Separator)) {
				rel += string(filepath.Separator)
			}

			return gi.MatchesPath(rel)
		}, nil
	}
}

func TruncateFilename(name string) string {
	return truncate.Truncate(name, 30, "...", truncate.PositionMiddle)
}

func FirstUpper(str string) string {
	if str == "" {
		return ""
	}

	return strings.ToUpper(str[0:1]) + str[1:]
}
