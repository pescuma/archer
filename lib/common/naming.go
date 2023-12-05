package common

import (
	"strings"

	"github.com/gertd/go-pluralize"

	"github.com/pescuma/archer/lib/model"
)

func CreateTableNameParts(projs []*model.Project) {
	pc := pluralize.NewClient()

	plurals := 0

	for _, proj := range projs {
		if pc.IsPlural(proj.Name) {
			plurals++
		}
	}

	convertToPlural := plurals > len(projs)/2

	for _, proj := range projs {
		pieces := strings.Split(proj.Name, "_")

		parts := make([]string, 0, len(pieces))

		for i := range pieces {
			part := strings.Join(pieces[:i+1], "_")

			if part == "" {
				continue
			}

			switch {
			case i == len(pieces)-1:
				part = proj.Name

			case convertToPlural:
				part = pc.Plural(part)

			default:
				part = pc.Singular(part)
			}

			parts = append(parts, part)
		}

		proj.Groups = parts
	}
}
