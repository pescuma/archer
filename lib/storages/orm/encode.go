package orm

import (
	"strings"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func encodeMetric(v int) *int {
	return utils.IIf(v == -1, nil, &v)
}
func decodeMetric(v *int) int {
	if v == nil {
		return -1
	} else {
		return *v
	}
}

func encodeMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) == 0 {
		return nil
	}

	return cloneMap(m)
}
func decodeMap[K comparable, V any](m map[K]V) map[K]V {
	return cloneMap(m)
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func encodeOldFileIDs(v map[model.ID]model.ID) string {
	var sb strings.Builder

	for k, v := range v {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(k.String())
		sb.WriteString(":")
		sb.WriteString(v.String())
	}

	return sb.String()
}
func decodeOldFileIDs(v string) map[model.ID]model.ID {
	result := make(map[model.ID]model.ID)
	if v == "" {
		return result
	}

	for _, line := range strings.Split(v, "\n") {
		cols := strings.Split(line, ":")
		result[model.MustStringToID(cols[0])] = model.MustStringToID(cols[1])
	}

	return result
}

func encodeOldFileHashes(v map[model.ID]string) string {
	var sb strings.Builder

	for k, v := range v {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(k.String())
		sb.WriteString(":")
		sb.WriteString(v)
	}

	return sb.String()
}
func decodeOldFileHashes(v string) map[model.ID]string {
	result := make(map[model.ID]string)
	if v == "" {
		return result
	}

	for _, line := range strings.Split(v, "\n") {
		cols := strings.Split(line, ":")
		result[model.MustStringToID(cols[0])] = cols[1]
	}

	return result
}
